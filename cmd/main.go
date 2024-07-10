package main

import (
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
)

var DEFAULT_LOGGER = slog.New(slog.NewTextHandler(os.Stderr, nil))

type ResponseHandler func(Request) *Response

// GET / HTTP/1.1
// Host: localhost:8080
// User-Agent: curl/8.6.0
// Accept: */*

type Request struct {
	Method  string
	Path    string
	Version string
	Headers map[string]string
	Body    string
	Conn    net.Conn
}

type Response struct {
	Code    int
	Headers map[string]string
	Body    string
}

func main() {
	slog.SetDefault(DEFAULT_LOGGER)

	socket, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		slog.Error("Error while listeningm", "ERROR", err)
	}

	for {
		conn, err := socket.Accept()
		if err != nil {
			slog.Error("Error while accepting connection", "ERROR", err)
		}

		go handleConnection(conn, func(r Request) *Response {
			return &Response{
				Code: 200,
				Headers: map[string]string{
					"Content-Type": "text/plain",
				},
				Body: "Hello, World!",
			}

		})

	}
}

func handleConnection(conn net.Conn, responseHandler ResponseHandler) {
	defer conn.Close()

	buf := make([]byte, 1024)
	_, err := conn.Read(buf)
	if err != nil {
		slog.Error("Error while reading from connection", "ERROR", err)
	}

	stringBuf := string(buf)
	var request Request

	sp := strings.Split(stringBuf, "\r\n")

	requestLine := strings.Split(sp[0], " ")
	request.Method = requestLine[0]
	request.Path = requestLine[1]
	request.Version = requestLine[2]

	request.Headers = make(map[string]string)
	for i := 1; i < len(sp); i++ {
		if sp[i] == "" {
			request.Body = sp[i+1]
			break
		}

		header := strings.Split(sp[i], ": ")
		request.Headers[header[0]] = header[1]
	}

	request.Conn = conn

	response := responseHandler(request)

	// check if header has a content-type
	if _, ok := response.Headers["Content-Type"]; !ok {
		response.Headers["Content-Type"] = "text/plain"
	}

	// add content length to Headers
	response.Headers["Content-Length"] = strconv.Itoa(len(response.Body))

	if response != nil {
		conn.Write([]byte(
			"HTTP/1.1 " + strconv.Itoa(response.Code) + "\r\n" +
				headerString(response.Headers) +
				"\r\n" +
				response.Body,
		))
	}
}

func headerString(headers map[string]string) string {
	var headerString string
	for key, value := range headers {
		headerString += key + ": " + value + "\r\n"
	}

	return headerString
}
