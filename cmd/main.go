package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/radenrishwan/hfs"
)

var DEFAULT_LOGGER = slog.New(slog.NewJSONHandler(os.Stderr, nil))
var websocket = hfs.NewWebsocket(nil)

func main() {
	slog.SetDefault(DEFAULT_LOGGER)

	server := hfs.NewServer("localhost:8080", hfs.Option{})
	defer server.Close()
	server.Use(func(r hfs.Request) {
		slog.Info("", "Version", r.Version, "Method", r.Method, "Path", r.Path, "Time", time.Now().String())
	})

	server.ServeDir("/", "html/")
	server.ServeFile("/", "html/index.html")

	server.Handle("/hello", func(r hfs.Request) *hfs.Response {
		return hfs.NewTextResponse("Hello, World")
	})

	server.Handle("/ws", func(r hfs.Request) *hfs.Response {
		client, err := websocket.Upgrade(r)
		defer client.Close("Closing connection", hfs.STATUS_CLOSE_NORMAL_CLOSURE)

		if err != nil {
			slog.Error("Error while upgrading to websocket", "ERROR", err)
			panic(hfs.NewHttpError(500, "error while upgrading ws", r))
		}

		for {
			msg, err := client.Read()
			if err != nil {
				slog.Error("Error while reading message", "ERROR", err)
				break
			}

			slog.Info("Message received", "Message", string(msg))

			err = client.Send(string(msg))
			if err != nil {
				slog.Error("Error while sending message", "ERROR", err)
				break
			}
		}

		return hfs.NewTextResponse("Hello, World")
	})

	server.ListenAndServe()
}
