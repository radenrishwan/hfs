package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/radenrishwan/hfs"
)

var DEFAULT_LOGGER = slog.New(slog.NewJSONHandler(os.Stderr, nil))
var websocket = hfs.NewWebsocket(nil)

func main() {
	slog.SetDefault(DEFAULT_LOGGER)

	server := hfs.NewServer("localhost:3000", hfs.Option{})

	server.SetErrHandler(func(req hfs.Request, err error) *hfs.Response {
		slog.Error("Error while handling request", "ERROR", err)

		if httpError, ok := err.(*hfs.HttpError); ok {
			if httpError.Code == http.StatusNotFound {
				return &hfs.Response{
					Code: 404,
					Headers: map[string]string{
						"Content-Type": "text/plain",
					},
					Body: "Not Found",
				}
			}
		}

		return &hfs.Response{
			Code: 500,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: "Internal Server Error",
		}
	})

	server.Handle("/ws", func(req hfs.Request) *hfs.Response {
		fmt.Println("Upgrading to websocket")
		client, err := websocket.Upgrade(req)
		if err != nil {
			panic(err)
		}

		for {
			p, err := client.Read()
			if err != nil {
				client.Close()
				slog.Error("Error while reading message", "ERROR", err)
				break
			}

			err = client.Send("Hello, Client")
			if err != nil {
				slog.Error("Error while sending message", "ERROR", err)
				break
			}

			fmt.Println("Received: ", string(p))
		}

		return &hfs.Response{
			Code: 200,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: "Websocket",
		}
	})

	server.Handle("GET /", func(req hfs.Request) *hfs.Response {
		// panic(hfs.NewHttpError(500, "Internal Server Error", req))

		return &hfs.Response{
			Code: 200,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: "Home",
		}
	})

	server.Handle("GET /cookie", func(req hfs.Request) *hfs.Response {
		fmt.Println(req.Cookie)

		response := hfs.NewResponse()
		response.SetCode(200)
		response.AddHeader("Content-Type", "text/plain")
		response.SetBody("Hello, World!")

		response.SetCookie("foo", "bar", "/", 3600)
		response.SetCookie("baz", "qux", "/", 3600)

		return response
	})

	server.Handle("/about", func(req hfs.Request) *hfs.Response {
		return &hfs.Response{
			Code: 200,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: "About Page",
		}
	})

	server.Handle("/args", func(req hfs.Request) *hfs.Response {
		response := hfs.NewResponse()
		response.SetCode(200)
		response.SetBody(req.Args["name"])

		return response
	})

	server.Handle("GET /get", func(req hfs.Request) *hfs.Response {
		response := hfs.NewResponse()
		response.SetCode(200)
		response.SetBody("GET")

		return response
	})

	server.Handle("POST /post", func(req hfs.Request) *hfs.Response {
		response := hfs.NewResponse()
		response.SetCode(200)
		response.SetBody("POST")

		return response
	})

	server.ListenAndServe()
}
