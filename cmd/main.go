package main

import (
	"log/slog"
	"os"

	hsp "github.com/radenrishwan/hfs"
)

var DEFAULT_LOGGER = slog.New(slog.NewTextHandler(os.Stderr, nil))

func main() {
	slog.SetDefault(DEFAULT_LOGGER)

	server := hsp.NewServer("localhost:3000", hsp.Option{})

	server.Handle("/", func(req hsp.Request) *hsp.Response {
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

	server.ListenAndServe()
}
