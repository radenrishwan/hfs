package hfs

import (
	"log/slog"
	"net"
	"net/http"
	"os"
)

type ResponseHandler func(Request) *Response
type ErrResponseHandler func(Request, error) *Response
type MiddlewareHandler func(Request)

type Handler struct {
	Path       string
	Method     string
	Handler    ResponseHandler
	Middleware []MiddlewareHandler
}

func (handler *Handler) Use(middleware MiddlewareHandler) *Handler {
	handler.Middleware = append(handler.Middleware, middleware)

	return handler
}

type Option struct {
	ErrHandler       ErrResponseHandler
	GlobalMiddleware []MiddlewareHandler
}

type Server struct {
	address  string
	socket   net.Listener
	Handlers []Handler
	Option   Option
}

func (s *Server) Use(middleware MiddlewareHandler) *Server {
	s.Option.GlobalMiddleware = append(s.Option.GlobalMiddleware, middleware)

	return s
}

func NewServer(address string, option Option) *Server {
	// check err handler in option is nil
	if option.ErrHandler == nil {
		option.ErrHandler = func(req Request, err error) *Response {
			slog.Error("Error while handling request", "ERROR", err)

			return &Response{
				Code: 500,
				Headers: map[string]string{
					"Content-Type": "text/plain",
				},
				Body: "Internal Server Error",
			}
		}
	}

	return &Server{
		address: address,
		Option:  option,
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

			response := s.Option.ErrHandler(request, NewServerError("Error while accepting connection"))
			writeResponse(response, conn)
		}

		if len(s.Handlers) == 0 {
			return NewServerError("No handler found for the request")
		}

		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	request, err := parseRequest(conn)
	if err != nil {
		response := s.Option.ErrHandler(Request{Conn: conn}, err)
		writeResponse(response, conn)
		return
	}

	var response *Response

	// find the handler for the request
	for _, handler := range s.Handlers {
		if request.Path == handler.Path {
			// check if method is not same, if method is "", call the handler instead
			if handler.Method != request.Method && handler.Method != "" {
				response = s.Option.ErrHandler(request, NewHttpError(405, "Method not allowed", request))
				break
			}

			func() {
				defer func() {
					rc := recover()

					// check if error is not nil
					if rc != nil {
						err = rc.(error)
					}

				}()

				// run global middleware
				for _, middleware := range s.Option.GlobalMiddleware {
					middleware(request)
				}

				// run middleware
				for _, middleware := range handler.Middleware {
					middleware(request)
				}

				response = handler.Handler(request)
			}()

			break
		}
	}

	if err != nil {
		response = s.Option.ErrHandler(request, err)
	}

	if response == nil {
		response = s.Option.ErrHandler(request, NewHttpError(404, "No handler found for the request", request))
	}

	writeResponse(response, conn)
}

// Handle registers a handler for the given path
// The middleware its different with global middleware, its not run for all request
func (s *Server) Handle(
	path string,
	handler ResponseHandler,
	middleware ...MiddlewareHandler,
) error {
	// check duplicate path
	for _, h := range s.Handlers {
		if h.Path == path {
			return NewServerError("Duplicate path found")
		}
	}

	method, path := parsePath(path)

	res := Handler{
		Path:       path,
		Handler:    handler,
		Method:     method,
		Middleware: make([]MiddlewareHandler, 0),
	}

	res.Middleware = append(res.Middleware, middleware...)

	s.Handlers = append(s.Handlers, res)

	return nil
}

func (s *Server) SetErrHandler(handler ErrResponseHandler) {
	s.Option.ErrHandler = handler
}

func (s *Server) ServeFile(path string, filePath string) error {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return NewServerError("Error while reading file: " + err.Error())
	}

	fileType := http.DetectContentType(file)

	return s.Handle(path, func(req Request) *Response {
		return &Response{
			Code: 200,
			Headers: map[string]string{
				"Content-Type": fileType,
			},
			Body: string(file),
		}
	})
}

// ServeFile serves a file at the given path
//
// server.ServeFile("GET /hello", "path/to/dir")
func (s *Server) ServeDir(prefixPath string, filePath string) error {
	// check last character of the path
	if filePath[len(filePath)-1] == '/' {
		filePath = filePath[:len(filePath)-1]
	}

	if prefixPath[len(prefixPath)-1] == '/' {
		prefixPath = prefixPath[:len(prefixPath)-1]
	}

	// get all files in the directory
	files, err := os.ReadDir(filePath)
	if err != nil {
		return NewServerError("Error while reading directory: " + err.Error())
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// read file
		output, err := os.ReadFile(filePath + "/" + file.Name())
		if err != nil {
			return NewServerError("Error while reading file: " + err.Error())
		}

		fileType := http.DetectContentType(output)

		err = s.Handle(prefixPath+"/"+file.Name(), func(req Request) *Response {
			return &Response{
				Code: 200,
				Headers: map[string]string{
					"Content-Type": fileType,
				},
				Body: string(output),
			}
		})

		if err != nil {
			return err
		}
	}

	return nil
}
