package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/alipourhabibi/http-downloader/http"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))
	slog.SetDefault(logger)
	args := os.Args[1:]
	if len(args) != 3 {
		os.Stderr.WriteString("./client [IPv4] [Port] [FileName]")
		return
	}

	port, err := strconv.Atoi(args[1])
	if err != nil {
		os.Stderr.WriteString("invalid port number")
		return
	}
	server, err := http.NewServer(args[0], port, 8000)

	var msg string
	msg = fmt.Sprintf("GET /%s HTTP/1.1\r\nHOST:localhost\r\nRange: bytes=\r\n\r\n", args[2])

	err = server.SendMsg(msg)
	_, response, err := server.RecieveMsg()

	status := http.GetStatus(response)
	if status == 404 {
		os.Stdout.WriteString("404 Not Found\n")
		return
	}
	if status >= 500 {
		os.Stdout.WriteString("Internal Server Error\n")
		return
	}
	header, err := http.GetHeader(response)
	if err != nil {
		os.Stdout.WriteString(err.Error())
		return
	}
	size := header["Content-Length"]
	intSize, err := strconv.Atoi(size)
	if err != nil {
		os.Stdout.WriteString(err.Error())
		return
	}
	server, err = http.NewServer(args[0], port, 8000)
	if intSize <= 1000 {
		err = server.DownloadOne(args[2], intSize)
		if err != nil {
			panic(err)
		}
	} else {
		if header["Accept-Ranges"] != "" && strings.HasPrefix(header["Accept-Ranges"], "bytes") {
			server.DownloadParallel(args[2], intSize)
		} else {
			server.DownloadOne(args[2], intSize)
		}
	}
}
