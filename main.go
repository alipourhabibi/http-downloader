package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"flag"

	"github.com/alipourhabibi/http-downloader/http"
)

var logLevel = flag.String("level", "error", "level is used to set the logging level")
var addSource = flag.Bool("source", false, "is used by logger to log the caller function or not")
var ip = flag.String("ip", "127.0.0.1", "destination ip of the server")
var port = flag.Int("port", 80, "destination port of the server")
var fileName = flag.String("filename", "", "file name which is going to be downloaded from server")

var levelMap = map[string]slog.Level{
	"DEBUG": slog.LevelDebug,
	"INFO":  slog.LevelInfo,
	"WARN":  slog.LevelWarn,
	"ERROR": slog.LevelError,
}

func main() {
	flag.Parse()

	level := levelMap[strings.ToUpper(*logLevel)]
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: *addSource,
		Level:     level,
	}))
	slog.SetDefault(logger)

	if *fileName == "" {
		slog.Error("filename", "error", "filename can't be empty")
		return
	}

	server, err := http.NewServer(*ip, *port, 8000)

	var msg string
	msg = fmt.Sprintf("GET /%s HTTP/1.1\r\nHOST:localhost\r\nRange: bytes=\r\n\r\n", *fileName)

	err = server.SendMsg(msg)
	_, response, err := server.RecieveMsg()
	if err != nil {
		slog.Error("RecieveMsg", "error", err.Error())
		return
	}

	status := http.GetStatus(response)
	if status == 404 {
		slog.Error("404 Not Found")
		return
	}
	if status >= 500 {
		slog.Error("Internal Server Error")
		return
	}
	header, err := http.GetHeader(response)
	if err != nil {
		slog.Error("GetHeader", "error", err.Error())
		return
	}
	size := header["Content-Length"]
	intSize, err := strconv.Atoi(size)
	if err != nil {
		slog.Error("strconv error", "error", err.Error())
		return
	}
	server, err = http.NewServer(*ip, *port, 8000)
	if intSize <= 1000 {
		err = server.DownloadOne(*fileName, intSize)
		if err != nil {
			panic(err)
		}
	} else {
		if header["Accept-Ranges"] != "" && strings.HasPrefix(header["Accept-Ranges"], "bytes") {
			server.DownloadParallel(*fileName, intSize)
		} else {
			server.DownloadOne(*fileName, intSize)
		}
	}
}
