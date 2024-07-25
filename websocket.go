package hsp

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
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

	msg := encodeFrame([]byte("hello world"))

	request.Conn.Write(msg)

	for {
		buf := make([]byte, 1024)

		n, err := request.Conn.Read(buf)
		if err != nil {
			fmt.Println("Error reading from connection", err)
			break
		}

		f, err := decodeFrame(buf[:n])
		if err != nil {
			fmt.Println("Error decoding frame", err)
			break
		}

		fmt.Println(string(f.Payload))
	}

	return &Client{
		Conn: &request.Conn,
	}
}

func generateWebsocketKey(key string) string {
	sha := sha1.New()
	sha.Write([]byte(key))
	sha.Write([]byte(MAGIC_KEY))

	return base64.StdEncoding.EncodeToString(sha.Sum(nil))
}

func encodeFrame(msg []byte) []byte {
	frame := make([]byte, 0)
	frame = append(frame, 0x81) // FIN, Opcode 1 (text frame)

	length := len(msg)
	if length < 126 {
		frame = append(frame, byte(length))
	} else if length <= 0xFFFF {
		frame = append(frame, 126)
		frame = append(frame, byte(length>>8))
		frame = append(frame, byte(length&0xFF))
	} else {
		frame = append(frame, 127)
		frame = append(frame, byte(length>>56))
		frame = append(frame, byte(length>>48&0xFF))
		frame = append(frame, byte(length>>40&0xFF))
		frame = append(frame, byte(length>>32&0xFF))
		frame = append(frame, byte(length>>24&0xFF))
		frame = append(frame, byte(length>>16&0xFF))
		frame = append(frame, byte(length>>8&0xFF))
		frame = append(frame, byte(length&0xFF))
	}

	frame = append(frame, msg...)
	return frame
}

func decodeFrame(data []byte) (*WsFrame, error) {
	if len(data) < 2 {
		return nil, NewWsError("insufficient data for frame", nil)
	}

	frame := &WsFrame{}
	// Decode the first byte (FIN, RSV1, RSV2, RSV3, opcode)
	frame.Fin = (data[0] & 0x80) != 0
	frame.RSV1 = (data[0] & 0x40) != 0
	frame.RSV2 = (data[0] & 0x20) != 0
	frame.RSV3 = (data[0] & 0x10) != 0
	frame.Opcode = data[0] & 0x0F

	// Decode the second byte (mask, payload length)
	frame.Mask = (data[1] & 0x80) != 0
	payloadLength := uint64(data[1] & 0x7F)

	// Determine the length of the payload and the position of the mask and payload
	var dataOffset uint64
	switch payloadLength {
	case 126:
		if len(data) < 4 {
			return nil, NewWsError("insufficient data for payload length", nil)
		}
		frame.Length = uint64(binary.BigEndian.Uint16(data[2:4]))
		dataOffset = 4
	case 127:
		if len(data) < 10 {
			return nil, NewWsError("insufficient data for payload length", nil)
		}
		frame.Length = binary.BigEndian.Uint64(data[2:10])
		dataOffset = 10
	default:
		frame.Length = payloadLength
		dataOffset = 2
	}

	// If the frame is masked, extract the mask key
	if frame.Mask {
		if uint64(len(data)) < dataOffset+4 {
			return nil, NewWsError("insufficient data for mask key", nil)
		}
		copy(frame.MaskKey[:], data[dataOffset:dataOffset+4])
		dataOffset += 4
	}

	// Extract the payload
	if uint64(len(data)) < dataOffset+frame.Length {
		return nil, NewWsError("insufficient data for payload", nil)
	}
	payload := data[dataOffset : dataOffset+frame.Length]

	// If the frame is masked, unmask the payload
	if frame.Mask {
		unmaskedPayload := make([]byte, len(payload))
		for i, b := range payload {
			unmaskedPayload[i] = b ^ frame.MaskKey[i%4]
		}
		payload = unmaskedPayload
	}

	frame.Payload = payload

	return frame, nil
}
