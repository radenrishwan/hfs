package hsp

import (
	"log/slog"
	"net"
	"strconv"
)

type ResponseHandler func(Request) *Response
type ErrResponseHandler func(Request, error)

type Request struct {
	Method  string
	Path    string
	Version string
	Headers map[string]string
	Cookie  map[string]string
	Body    string
	Conn    net.Conn
}

type Response struct {
	Code    int
	Headers map[string]string
	Body    string
}

func NewResponse() *Response {
	return &Response{
		Code:    200,
		Headers: make(map[string]string),
	}
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

func (r *Response) SetCookie(key, value, path string, maxAge int) {
	r.Headers["Set-Cookie"] = key + "=" + value + "; Path=" + path + "; Max-Age=" + strconv.Itoa(maxAge)
}

type Handler struct {
	Path    string
	Handler ResponseHandler
}

type Server struct {
	address    string
	socket     net.Listener
	Handlers   []Handler
	ErrHandler ErrResponseHandler
}

type Option struct {
	ErrHandler ErrResponseHandler
}

func NewServer(address string, option Option) *Server {
	// check err handler in option is nil
	if option.ErrHandler == nil {
		option.ErrHandler = func(req Request, err error) {
			slog.Error("Error while handling request", "ERROR", err)

			response := Response{
				Code: 500,
				Headers: map[string]string{
					"Content-Type": "text/plain",
				},
				Body: "Internal Server Error",
			}

			writeResponse(&response, req.Conn)
		}
	}

	return &Server{
		address:    address,
		ErrHandler: option.ErrHandler,
	}
}

func (s *Server) ListenAndServe() error {
	socket, err := net.Listen("tcp", s.address)
	if err != nil {
		return NewServerError("Error while listening to address: " + err.Error())
	}

	for {
		conn, err := socket.Accept()
		if err != nil {
			request := Request{
				Conn:    conn,
				Path:    "",
				Method:  "",
				Version: "",
				Headers: make(map[string]string),
				Cookie:  make(map[string]string),
				Body:    "Server Error",
			}

			s.ErrHandler(request, NewServerError("Error while accepting connection"))
		}

		if len(s.Handlers) == 0 {
			return NewServerError("No handler found for the request")
		}

		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	request := parseRequest(conn)
	var err error
	var response *Response

	// find the handler for the request
	for _, handler := range s.Handlers {
		if request.Path == handler.Path {
			func() {
				defer func() {
					rc := recover()

					// check if error is not nil
					if rc != nil {
						err = rc.(error)
					}

				}()

				response = handler.Handler(request)
			}()
		}
	}

	if err != nil {
		s.ErrHandler(request, err)
	}

	if response == nil {
		s.ErrHandler(request, NewHttpError(404, "No handler found for the request", request))
		return
	}

	writeResponse(response, conn)
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

func (s *Server) SetErrHandler(handler ErrResponseHandler) {
	s.ErrHandler = handler
}
