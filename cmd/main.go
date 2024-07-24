package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	hsp "github.com/radenrishwan/hfs"
)

var DEFAULT_LOGGER = slog.New(slog.NewTextHandler(os.Stderr, nil))
var websocket = hsp.NewWebsocket()

func main() {
	slog.SetDefault(DEFAULT_LOGGER)

	server := hsp.NewServer("localhost:3000", hsp.Option{})

	server.SetErrHandler(func(req hsp.Request, err error) *hsp.Response {
		slog.Error("Error while handling request", "ERROR", err)

		if httpError, ok := err.(*hsp.HttpError); ok {
			if httpError.Code == http.StatusNotFound {
				return &hsp.Response{
					Code: 404,
					Headers: map[string]string{
						"Content-Type": "text/plain",
					},
					Body: "Not Found",
				}
			}
		}

		return &hsp.Response{
			Code: 500,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: "Internal Server Error",
		}
	})

	server.Handle("/ws", func(req hsp.Request) *hsp.Response {
		// print headers
		// fmt.Println(req.Headers)

		fmt.Println("Upgrading to websocket")
		websocket.Upgrade(req)

		return &hsp.Response{
			Code: 200,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: "Websocket",
		}
	})

	server.Handle("/", func(req hsp.Request) *hsp.Response {
		// panic(hsp.NewHttpError(500, "Internal Server Error", req))

		return &hsp.Response{
			Code: 200,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: "Home",
		}
	})

	server.Handle("GET /cookie", func(req hsp.Request) *hsp.Response {
		fmt.Println(req.Cookie)

		response := hsp.NewResponse()
		response.SetCode(200)
		response.AddHeader("Content-Type", "text/plain")
		response.SetBody("Hello, World!")

		response.SetCookie("foo", "bar", "/", 3600)
		response.SetCookie("baz", "qux", "/", 3600)

		return response
	})

	server.Handle("/about", func(req hsp.Request) *hsp.Response {
		return &hsp.Response{
			Code: 200,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: "About Page",
		}
	})

	server.Handle("/args", func(req hsp.Request) *hsp.Response {
		response := hsp.NewResponse()
		response.SetCode(200)
		response.SetBody(req.Args["name"])

		return response
	})

	server.Handle("GET /get", func(req hsp.Request) *hsp.Response {
		response := hsp.NewResponse()
		response.SetCode(200)
		response.SetBody("GET")

		return response
	})

	server.Handle("POST /post", func(req hsp.Request) *hsp.Response {
		response := hsp.NewResponse()
		response.SetCode(200)
		response.SetBody("POST")

		return response
	})

	server.ListenAndServe()
}
