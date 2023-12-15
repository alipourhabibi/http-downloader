package http

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/alipourhabibi/http-downloader/writer"
	"golang.org/x/exp/slices"
)

type server struct {
	maxSize    int64
	fd         int
	serverAddr *syscall.SockaddrInet4
}

func NewServer(host string, port int, maxSize int64) (*server, error) {
	serverFD, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_IP)
	if err != nil {
		slog.Error("Socker: ", "error", err.Error())
		os.Exit(1)
	}

	if err != nil || (port < 0 || port > 65535) {
		slog.Error("Invalid port", "port", port)
		return nil, err
	}
	serverAddr := &syscall.SockaddrInet4{
		Port: port,
		Addr: inetAddr(host),
	}

	err = syscall.Connect(serverFD, serverAddr)
	if err != nil {
		if err == syscall.ECONNREFUSED {
			slog.Error("* connection falied")
			syscall.Close(serverFD)
			return nil, err
		}
	}
	return &server{
		fd:         serverFD,
		serverAddr: serverAddr,
		maxSize:    maxSize,
	}, nil
}

func (s *server) SendMsg(msg string) error {
	var err error
	err = syscall.Sendmsg(
		s.fd,
		[]byte(msg),
		nil, s.serverAddr, syscall.MSG_WAITALL)
	if err != nil {
		return err
	}
	return nil
}
func (s *server) RecieveMsgFlag(flag, size int) (int, []byte, error) {
	response := make([]byte, size)
	n, _, err := syscall.Recvfrom(s.fd, response, flag)
	if err != nil {
		syscall.Close(s.fd)
		return 0, nil, err
	}
	return n, response, nil
}

func (s *server) RecieveMsg() (int, []byte, error) {
	response := make([]byte, s.maxSize)
	n, _, err := syscall.Recvfrom(s.fd, response, syscall.MSG_WAITFORONE)
	if err != nil {
		syscall.Close(s.fd)
		return 0, nil, err
	}
	return n, response, nil
}

func (s *server) DownloadParallel(resourse string, length int) error {
	slog.Info("DonwloadParallel", "resourse", resourse, "size", length)
	err := writer.Create(resourse)
	if err != nil {
		return err
	}
	wg := sync.WaitGroup{}
	size := int(length / 10)
	offset := 0
	for offset = 0; offset < 10; offset++ {
		wg.Add(1)
		if offset == 9 {
			go save(s.fd, s.serverAddr, resourse, offset, size, size+(length-(size*10)), &wg)
		} else {
			go save(s.fd, s.serverAddr, resourse, offset, size, size, &wg)
		}
	}
	// wg.Add(1)
	// save(s.fd, s.serverAddr, resourse, offset, size, length-(size*10)+size, &wg)
	wg.Wait()
	return nil
}

func (s *server) DownloadOne(resourse string, size int) error {
	slog.Info("DonwloadOne", "resourse", resourse, "size", size)
	err := writer.Create(resourse)
	if err != nil {
		return err
	}
	msg := fmt.Sprintf("GET /%s HTTP/1.1\r\nConnection: keep-alive\r\nHOST: localhost\r\n\r\n", resourse)
	err = s.SendMsg(msg)
	if err != nil {
		return err
	}
	n, resp, err := s.RecieveMsg()
	if err != nil {
		return err
	}
	index := 0
	w := writer.NewFile(resourse)
	var header []byte
	if strings.Contains(string(resp), "\r\n\r\n") {
		header = []byte(strings.Split(string(resp), "\r\n\r\n")[0])
		resp = []byte(strings.Split(string(resp), "\r\n\r\n")[1])
		resp = resp[:n-len(header)]
		if !slices.Equal(resp, []byte{0, 0, 0, 0}) {
			w.Save(resp, int64(index))
			index += n - len(header) - 4
		}
	}
	for {
		n, resp, err = s.RecieveMsg()
		if err != nil {
			return err
		}
		if n == 0 {
			break
		}
		w.Save(resp[:n], int64(index))
		index += n
	}
	w.Close()
	return nil
}

func (s *server) Close() error {
	return syscall.Close(s.fd)
}

func inetAddr(ipaddr string) [4]byte {
	var (
		ip                 = strings.Split(ipaddr, ".")
		ip1, ip2, ip3, ip4 uint64
	)
	ip1, _ = strconv.ParseUint(ip[0], 10, 8)
	ip2, _ = strconv.ParseUint(ip[1], 10, 8)
	ip3, _ = strconv.ParseUint(ip[2], 10, 8)
	ip4, _ = strconv.ParseUint(ip[3], 10, 8)
	return [4]byte{byte(ip1), byte(ip2), byte(ip3), byte(ip4)}
}

func GetHeader(response []byte) (map[string]string, error) {
	if strings.Contains(string(response), "\r\n\r\n") {
		header := strings.Split(string(response), "\r\n")[1:]
		mapHeader := map[string]string{}
		for _, v := range header {
			h := strings.Split(v, ":")
			if len(h) == 2 {
				// Remove the first space
				mapHeader[h[0]] = h[1][1:]
			}
		}
		return mapHeader, nil
	} else {
		return nil, fmt.Errorf("No header")
	}
}

func GetStatus(response []byte) int {
	f := strings.Split(string(response), "\n")[0]
	f = strings.Split(f, " ")[1]
	i, _ := strconv.Atoi(f)
	return i
}

func save(serverFD int, serverAddr *syscall.SockaddrInet4, file string, offset, msgSize, size int, wg *sync.WaitGroup) {
	slog.Info("Partial Download", "file", file, "offset", offset, "size", size)
	serverFD, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_IP)
	err = syscall.Connect(serverFD, serverAddr)
	if err != nil {
		if err == syscall.ECONNREFUSED {
			slog.Error("* Connection failed")
			syscall.Close(serverFD)
			return
		}
	}

	b := offset * msgSize
	w := writer.NewFile(file)
	defer w.Close()
	var msg string
	msg = fmt.Sprintf("GET /%s HTTP/1.1\r\nHOST: localhost\r\nConnection: keep-alive\r\nRange: bytes=%d-%d\r\n\r\n", file, b, b+size)
	slog.Debug("request msg", "msg", msg)
	err = syscall.Sendmsg(
		serverFD,
		[]byte(msg),
		nil, serverAddr, syscall.MSG_WAITFORONE)
	if err != nil {
		slog.Error("Sendmsg: ", "error", err.Error())
		return
	}
	response := make([]byte, 8000)
	n, _, err := syscall.Recvfrom(serverFD, response, syscall.MSG_WAITFORONE)
	if err != nil {
		slog.Error("Recvfrom: ", "error", err.Error())
		syscall.Close(serverFD)
		return
	}
	index := 0
	if strings.Contains(string(response), "\r\n\r\n") {
		splitedResp := strings.Split(string(response), "\r\n\r\n")
		if len(splitedResp) >= 2 {
			header := []byte(splitedResp[0])
			response = []byte(strings.Join(splitedResp[1:], "\r\n\r\n"))
			response = response[:n-len(header)]
			response = response[:len(response)-4]
			if len(response) != 0 {
				w.Save(response, int64(b))
				index += len(response)
				b += index
			}
		} else {
			wg.Done()
			slog.Info("save done", "offset", offset)
			return
		}
	}

	for {
		if index >= msgSize {
			break
		}
		response = make([]byte, 8000)
		n, _, err := syscall.Recvfrom(serverFD, response, syscall.MSG_WAITFORONE)
		if err != nil {
			slog.Error("Recvfrom: ", "error", err.Error())
			syscall.Close(serverFD)
			return
		}
		w.Lock()
		w.Save(response[:n], int64(b))
		w.Unlock()
		index += n
		b += n
	}
	wg.Done()
	slog.Info("save done", "offset", offset)
}
