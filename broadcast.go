package hfs

type Room struct {
	Client []*Client
}

func newRoom() (room Room) {
	return Room{
		Client: []*Client{},
	}
}

func (ws *Websocket) CreateRoom(name string) error {
	// check if room already exists
	if _, ok := ws.Rooms.Load(name); ok {
		return NewWsError("Room already exists")
	}

	room := newRoom()

	ws.Rooms.Store(name, &room)

	return nil
}

func (ws *Websocket) GetRoom(name string) (*Room, bool) {
	room, ok := ws.Rooms.Load(name)

	r, rok := room.(*Room)
	if !rok {
		return nil, false
	}

	return r, ok
}

func (ws *Websocket) RemoveRoom(name string) {
	ws.Rooms.Delete(name)
}

func (ws *Websocket) Broadcast(roomName string, msg string, ignoreError bool) error {
	room, ok := ws.GetRoom(roomName)
	if !ok {
		return NewWsError("Room not found")
	}

	for _, client := range room.Client {
		err := client.SendWithMessageType(msg, TEXT)
		// check if client is still connected
		if err != nil {
			room.RemoveClient(client)
		}

		if !ignoreError {
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (ws *Websocket) BroadcastBytes(roomName string, msg []byte, ignoreError bool) error {
	room, ok := ws.GetRoom(roomName)
	if !ok {
		return NewWsError("Room not found")
	}

	for _, client := range room.Client {
		err := client.SendBytes(msg)

		if !ignoreError {
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (ws *Websocket) BroadcastWithMessageType(roomName string, msg string, msgType MessageType, ignoreError bool) error {
	room, ok := ws.GetRoom(roomName)
	if !ok {
		return NewWsError("Room not found")
	}

	for _, client := range room.Client {
		err := client.SendWithMessageType(msg, msgType)

		if !ignoreError {
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (ws *Websocket) GetRoomList() []string {
	// get all room
	rooms := make([]string, 0)

	ws.Rooms.Range(func(key, value any) bool {
		rooms = append(rooms, key.(string))
		return true

	})

	return rooms
}

func (room *Room) AddClient(client *Client) {
	room.Client = append(room.Client, client)
}

func (room *Room) RemoveClient(client *Client) {
	for i, c := range room.Client {
		if c == client {
			room.Client = append(room.Client[:i], room.Client[i+1:]...)
			break
		}
	}
}

func (room *Room) GetClientCount() int {
	return len(room.Client)
}
