package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/radenrishwan/hfs"
)

var DEFAULT_LOGGER = slog.New(slog.NewJSONHandler(os.Stderr, nil))
var websocket = hfs.NewWebsocket(nil)

const (
	MSG        = 0
	JOIN       = 1
	LEAVE      = 2
	PRIVATEMSG = 3
	TYPING     = 4
	STOPTYPING = 5
)

type User struct {
	Id   string
	Name string
	Conn hfs.Client
}

func NewUser(id string, name string, conn hfs.Client) *User {
	return &User{
		Id:   id,
		Name: name,
		Conn: conn,
	}
}

type Message struct {
	Type    int    `json:"type"`
	Content string `json:"content"`
	From    string `json:"from"`
}

func NewMessage(t int, c string, f string) *Message {
	return &Message{
		Type:    t,
		Content: c,
		From:    f,
	}
}

func (m *Message) ToJson() string {
	output, _ := json.Marshal(m)

	return string(output)
}

type UserMessage struct {
	Type    int    `json:"type"`
	Content string `json:"content"`
}

func NewUserMessage() *UserMessage {
	return &UserMessage{}
}

func (m *UserMessage) FromJson(data string) error {
	return json.Unmarshal([]byte(data), m)
}

type Pool struct {
	Register   chan *User
	Unregister chan *User
	Message    chan *Message
	Users      map[string]*User
}

func NewPool() *Pool {
	return &Pool{
		Register:   make(chan *User),
		Unregister: make(chan *User),
		Message:    make(chan *Message),
		Users:      make(map[string]*User), // TODO: add r/w mutex
	}
}

func (p *Pool) ListUsers() string {
	result := ""
	for _, u := range p.Users {
		result += fmt.Sprintf("%s-%s\n", u.Id, u.Name)
	}

	return result
}

func (p *Pool) Start() {
	for {
		select {
		case user := <-p.Register:
			p.Users[user.Id] = user
			for _, u := range p.Users {
				u.Conn.Send(NewMessage(JOIN, p.ListUsers(), user.Name).ToJson())
			}
			break
		case user := <-p.Unregister:
			delete(p.Users, user.Id)
			for _, u := range p.Users {
				u.Conn.Send(NewMessage(LEAVE, p.ListUsers(), user.Name).ToJson())
			}
		case message := <-p.Message:
			for _, u := range p.Users {
				u.Conn.Send(message.ToJson())
			}
		}
	}
}

func main() {
	pool := NewPool()
	slog.SetDefault(DEFAULT_LOGGER)

	go pool.Start()
	server := hfs.NewServer("localhost:8080", hfs.Option{})

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

	server.Handle("/ws", func(req hfs.Request) *hfs.Response {
		user := User{
			Id: strconv.Itoa(int(time.Now().UnixNano())),
		}
		user.Name = req.GetArgs("name")
		if user.Name == "" {
			user.Name = user.Id
		}

		client, err := websocket.Upgrade(req)
		user.Conn = client
		if err != nil {
			slog.Error("Error while upgrading to websocket", "ERROR", err)
			panic(hfs.NewHttpError(http.StatusInternalServerError, "error while upgrading ws", req))
		}
		pool.Register <- &user

		client.Send(NewMessage(PRIVATEMSG, "", user.Name).ToJson())

		for {
			msg, err := client.Read()
			if err != nil {
				client.Close()
				pool.Unregister <- &user
				slog.Error("Error while reading message on read", "ERROR", err)
				break
			}

			fmt.Println("Message from client:", string(msg))

			userMSG := NewUserMessage()
			err = userMSG.FromJson(string(msg))
			if err != nil {
				slog.Error("Error while unmarshalling message", "ERROR", err)
				continue
			}

			switch userMSG.Type {
			case TYPING:
				pool.Message <- NewMessage(TYPING, "", user.Name)
			case STOPTYPING:
				pool.Message <- NewMessage(STOPTYPING, "", user.Name)
			default:
				pool.Message <- NewMessage(MSG, userMSG.Content, user.Name)
			}
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
		html, err := os.ReadFile("html/chat.html")
		if err != nil {
			slog.Error("Error while reading file", "ERROR", err)

			return hfs.NewTextResponse("Error bang")
		}

		return hfs.NewHTMLResponse(string(html))
	})

	err := server.ListenAndServe()
	if err != nil {
		slog.Error("Error while listening and serving", "ERROR", err)
	}
}
