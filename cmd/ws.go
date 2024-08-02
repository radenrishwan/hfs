package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/radenrishwan/hfs"
)

type MsgType string

const (
	JOIN  MsgType = "JOIN"
	LEAVE MsgType = "LEAVE"
	MSG   MsgType = "MSG"
)

type Msg struct {
	Type MsgType
	Body string
	From User
}

type User struct {
	Name string
	Conn hfs.Client
}

type Room struct {
	Name  string
	Users map[string]User
	Msg   chan Msg
	Leave chan Msg
	Join  chan Msg
}

func (room *Room) Add(user User) {
	room.Users[user.Name] = user

	room.Broadcast(Msg{
		Type: JOIN,
		Body: user.Name + " has joined",
		From: user,
	})
}

func (room *Room) Remove(name string) {
	delete(room.Users, name)

	room.Broadcast(Msg{
		Type: LEAVE,
		Body: name + " has left",
		From: room.Users[name],
	})
}

func (room *Room) Broadcast(msg Msg) {
	for _, c := range room.Users {
		if msg.From.Name == c.Name {
			continue
		}
		c.Conn.Send(msg.From.Name + ": " + msg.Body)
	}
}

func (room *Room) Pool() {
	fmt.Println("Running room Pool: ", room.Name)
	for {
		select {
		case msg := <-room.Msg:
			room.Broadcast(msg)
		case msg := <-room.Leave:
			room.Remove(msg.From.Name)
		case msg := <-room.Join:
			room.Add(msg.From)
		}
	}
}

var ws = hfs.NewWebsocket(&hfs.DefaultWSOption)

func main() {
	server := hfs.NewServer("localhost:8080", hfs.Option{})

	var room = Room{
		Name:  "General",
		Users: make(map[string]User),
		Msg:   make(chan Msg),
		Leave: make(chan Msg),
		Join:  make(chan Msg),
	}
	go room.Pool()

	server.Handle("GET /", func(req hfs.Request) *hfs.Response {
		n, err := os.ReadFile("chat.html")
		if err != nil {
			return &hfs.Response{
				Code: 500,
				Body: "Failed reading index.html",
			}
		}

		resp := hfs.NewResponse()
		resp.Code = 200
		resp.Headers["Content-Type"] = "text/html"
		resp.Body = string(n)

		return resp
	})

	server.Handle("GET /ws", func(req hfs.Request) *hfs.Response {
		client, err := ws.Upgrade(req)
		if err != nil {
			return &hfs.Response{
				Code: 500,
				Body: "Failed upgrading to websocket",
			}
		}

		user := User{
			Name: strconv.Itoa(time.Now().Nanosecond()),
			Conn: client,
		}

		client.Send("Welcome to the chat room")

		room.Join <- Msg{
			Type: JOIN,
			Body: "",
			From: user,
		}

		slog.Info("User joined", "USER", user)
		for {
			msg, err := client.Read()
			if err != nil {
				slog.Error("Error while reading message", "ERROR", err)
				room.Leave <- Msg{
					Type: LEAVE,
					Body: "",
					From: user,
				}
				break
			}

			room.Msg <- Msg{
				Type: MSG,
				Body: string(msg),
				From: user,
			}
		}

		return &hfs.Response{
			Code: 200,
			Body: "OK",
		}
	})

	if err := server.ListenAndServe(); err != nil {
		slog.Error("Error while listening to address", "ERROR", err)
	}
}
