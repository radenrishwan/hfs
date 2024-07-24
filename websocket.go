package hsp

import (
	"fmt"
	"net"
)

const MAGIC_KEY = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

type Websocket struct{}

type Client struct {
	Id   string
	Conn *net.Conn
}

type WsFrame struct {
	Fin     bool
	RSV1    bool
	RSV2    bool
	RSV3    bool
	Opcode  uint8
	Mask    bool
	Length  uint64
	MaskKey [4]byte
	Payload []byte
}

func NewWebsocket() *Websocket {
	return &Websocket{}
}

// HTTP/1.1 101 Switching Protocols
// Upgrade: websocket
// Connection: Upgrade
// Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=
// Sec-WebSocket-Protocol: chat

func (ws *Websocket) Upgrade(request Request) *Client {
	key := request.Headers["Sec-WebSocket-Key"]

	acceptKey := generateWebsocketKey(key)

	request.Conn.Write([]byte(
		"HTTP/1.1 101 Switching Protocols\r\n" +
			"Upgrade: websocket\r\n" +
			"Connection: Upgrade\r\n" +
			"Sec-WebSocket-Accept: " + acceptKey + "\r\n" +
			"\r\n",
	))

	for {
		buf := make([]byte, 1024)

		n, err := request.Conn.Read(buf)
		if err != nil {
			fmt.Println("Error reading from connection", err)
			break
		}

		fmt.Println("Received", n, "bytes")
	}

	return &Client{
		Conn: &request.Conn,
	}
}
