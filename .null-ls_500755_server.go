package hsp

import (
	"log/slog"
	"net"
	"strconv"
)

type ResponseHandler func(Request) *Response

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

var error chan error

func (s *Server) ListenAndServer() error {
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
