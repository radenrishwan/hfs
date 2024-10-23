package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/radenrishwan/hfs"
)

var DEFAULT_LOGGER = slog.New(slog.NewJSONHandler(os.Stderr, nil))
var websocket = hfs.NewWebsocket(nil)

func main() {
	server := hfs.NewServer("localhost:8080", hfs.Option{})
	defer server.Close()

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

			return &hfs.Response{
				Code: httpError.Code,
				Headers: map[string]string{
					"Content-Type": "text/plain",
				},
				Body: httpError.Msg,
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

	server.ServeFile("/", "html/simple_ws.html")

	server.Handle("/ws", func(req hfs.Request) *hfs.Response {
		// get name and room from query string
		// name := req.GetArgs("name")
		room := req.GetArgs("room")

		client, err := websocket.Upgrade(req)
		slog.Error("ERROR", err)

		// add to broadcast
		websocket.CreateRoom(room)

		// add to room
		b, _ := websocket.GetRoom(room)
		b.AddClient(&client)

		for {
			msg, err := client.Read()
			if err != nil {
				slog.Error("Error while reading message", "ERROR", err)
				break
			}

			slog.Info("MSG", string(msg))

			websocket.Broadcast(room, string("MSG INCOMMING : "+string(msg)), true)
		}

		return hfs.NewTextResponse("OK")
	})

	slog.Info("Server running...")
	err := server.ListenAndServe()
	if err != nil {
		slog.Error("Error while listening and serving", "ERROR", err)
	}
}
