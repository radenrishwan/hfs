package hfs

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"net"
	"time"
)

const MAGIC_KEY = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

type Websocket struct {
	Option *WSOption
}

type Client struct {
	Id     int64
	Conn   net.Conn
	option *WSOption
}

type WSOption struct {
	MsgMaxSize int
}

var DefaultWSOption = WSOption{
	MsgMaxSize: 1024,
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

func NewWebsocket(option *WSOption) (ws Websocket) {
	if option == nil {
		option = &DefaultWSOption
	}

	return Websocket{
		Option: option,
	}
}

// HTTP/1.1 101 Switching Protocols
// Upgrade: websocket
// Connection: Upgrade
// Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=
// Sec-WebSocket-Protocol: chat

func (ws *Websocket) Upgrade(request Request) (client Client, err error) {
	key := request.Headers["Sec-WebSocket-Key"]
	if key == "" {
		return client, NewWsError("Sec-WebSocket-Key is required")
	}

	acceptKey := generateWebsocketKey(key)

	_, err = request.Conn.Write([]byte(
		"HTTP/1.1 101 Switching Protocols\r\n" +
			"Upgrade: websocket\r\n" +
			"Connection: Upgrade\r\n" +
			"Sec-WebSocket-Accept: " + acceptKey + "\r\n" +
			"\r\n",
	))

	if err != nil {
		return client, NewWsError("Error while upgrading connection : " + err.Error())
	}

	client.Conn = request.Conn
	client.Id = time.Now().UnixNano()
	client.option = ws.Option

	return client, nil
}

func (client *Client) Send(msg string) error {
	frame := encodeFrame([]byte(msg))

	_, err := client.Conn.Write(frame)
	if err != nil {
		return NewWsError("Error sending message : " + err.Error())
	}

	return nil
}

func (client *Client) Read() ([]byte, error) {
	buf := make([]byte, client.option.MsgMaxSize)

	n, err := client.Conn.Read(buf)
	if err != nil {
		return nil, NewWsError("Error reading message : " + err.Error())
	}

	f, err := decodeFrame(buf[:n])
	if err != nil {
		return nil, NewWsError("Error decoding frame : " + err.Error())
	}

	return f.Payload, nil
}

func (client *Client) Close() error {
	// TODO: Send close signal
	err := client.Conn.Close()
	if err != nil {
		return NewWsError("Error closing connection : " + err.Error())
	}

	return nil
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

		// add length as 16-bit unsigned integer
		frame = append(frame, byte(length>>8))
		frame = append(frame, byte(length&0xFF))
	} else {
		frame = append(frame, 127)

		// add length as 64-bit unsigned integer
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
		return nil, NewWsError("insufficient data for frame")
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
			return nil, NewWsError("insufficient data for payload length")
		}
		frame.Length = uint64(binary.BigEndian.Uint16(data[2:4]))
		dataOffset = 4
	case 127:
		if len(data) < 10 {
			return nil, NewWsError("insufficient data for payload length")
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
			return nil, NewWsError("insufficient data for mask key")
		}
		copy(frame.MaskKey[:], data[dataOffset:dataOffset+4])
		dataOffset += 4
	}

	// Extract the payload
	if uint64(len(data)) < dataOffset+frame.Length {
		return nil, NewWsError("insufficient data for payload")
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