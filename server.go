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

func (r *Response) AddHeader(key, value string) {
	r.Headers[key] = value
}

func (r *Response) SetBody(body string) {
	r.Body = body
}

func (r *Response) SetCode(code int) {
	r.Code = code
}

type Handler struct {
	Path    string
	Handler ResponseHandler
}

type Server struct {
	address    string
	socket     net.Listener
	Handlers   []Handler
	ErrHandler func(error)
}

type Option struct {
	ErrHandler func(error)
}

func NewServer(address string, option Option) *Server {
	// check err handler in option is nil
	if option.ErrHandler == nil {
		option.ErrHandler = func(err error) {
			slog.Error("Error while handling request", "ERROR", err)
		}
	}

	return &Server{
		address:    address,
		ErrHandler: option.ErrHandler,
	}
}

var httpErr = make(chan error)

func (s *Server) handleError() {
	for {
		select {
		case err := <-httpErr:
			s.ErrHandler(err)
		default:
			continue
		}
	}
}

func (s *Server) ListenAndServe() error {
	socket, err := net.Listen("tcp", s.address)
	if err != nil {
		return NewServerError("Error while listening to address: " + err.Error())
	}

	go s.handleError()

	for {
		conn, err := socket.Accept()
		if err != nil {
			httpErr <- NewHandlingError("Error while accepting connection")
		}

		if len(s.Handlers) == 0 {
			httpErr <- NewHandlingError("No handler found for the request")
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
		httpErr <- NewHttpError("404", "No handler found for the request")
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

func (s *Server) Handle(path string, handler ResponseHandler) error {
	// check duplicate path
	for _, h := range s.Handlers {
		if h.Path == path {
			return NewServerError("Duplicate path found")
		}
	}

	s.Handlers = append(s.Handlers, Handler{
		Path:    path,
		Handler: handler,
	})

	return nil
}

func (s *Server) SetErrHandler(handler func(error)) {
	s.ErrHandler = handler
}
