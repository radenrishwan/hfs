package hsp

import (
	"log/slog"
	"net"
)

type ResponseHandler func(Request) *Response
type ErrResponseHandler func(Request, error)

type Handler struct {
	Path    string
	Method  string
	Handler ResponseHandler
}

type Option struct {
	ErrHandler ErrResponseHandler
}

type Server struct {
	address    string
	socket     net.Listener
	Handlers   []Handler
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

  method, path := parsePath(path)

  if 

	s.Handlers = append(s.Handlers, Handler{
		Path:    path,
		Handler: handler,
    Method: method
		// TODO: add method later
	})

	return nil
}

func (s *Server) SetErrHandler(handler ErrResponseHandler) {
	s.ErrHandler = handler
}
