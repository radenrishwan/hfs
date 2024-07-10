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

type Handler struct {
	Path    string
	Handler ResponseHandler
}

type Server struct {
	address  string
	socket   net.Listener
	Handlers []Handler
}

func NewServer(address string) *Server {
	return &Server{
		address: address,
	}
}

func (s *Server) ListenAndServer() {
	socket, err := net.Listen("tcp", s.address)
	if err != nil {
		slog.Error("Error while listening", "ERROR", err)
	}

	for {
		conn, err := socket.Accept()
		if err != nil {
			slog.Error("Error while accepting connection", "ERROR", err)
		}

		if len(s.Handlers) == 0 {
			slog.Error("No handlers registered")
		}

		go s.handleConnection(conn)
	}
}

func parseRequest(conn net.Conn) Request {
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

	return request
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	request := parseRequest(conn)
	var response *Response

	// find the handler for the request
	for _, handler := range s.Handlers {
		if request.Path == handler.Path {
			response = handler.Handler(request)
			break
		}
	}

	if response == nil {
		slog.Error("No handler found for the request")
		return
	}

	// check if header has a content-type
	if _, ok := response.Headers["Content-Type"]; !ok {
		response.Headers["Content-Type"] = "text/plain"
	}

	// add content length to Headers
	response.Headers["Content-Length"] = strconv.Itoa(len(response.Body))

	// check if code is 0
	if response.Code == 0 {
		response.Code = 200
	}

	conn.Write([]byte(
		"HTTP/1.1 " + strconv.Itoa(response.Code) + "\r\n" +
			headerString(response.Headers) +
			"\r\n" +
			response.Body,
	))
}

func headerString(headers map[string]string) string {
	var headerString string
	for key, value := range headers {
		headerString += key + ": " + value + "\r\n"
	}

	return headerString
}

func (s *Server) Handle(path string, handler ResponseHandler) {
	// check duplicate path
	for _, h := range s.Handlers {
		if h.Path == path {
			slog.Error("Duplicate path", "PATH", path)
		}

	}

	s.Handlers = append(s.Handlers, Handler{
		Path:    path,
		Handler: handler,
	})
}

func main() {
	slog.SetDefault(DEFAULT_LOGGER)

	server := NewServer("localhost:3000")

	server.Handle("/", func(req Request) *Response {
		return &Response{
			Code: 200,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: "Hello, World!",
		}
	})

	server.Handle("/about", func(req Request) *Response {
		return &Response{
			Code: 200,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: "About Page",
		}
	})

	server.ListenAndServer()
}
