package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

type websocketMessage struct {
	Event  string `json:"event"`
	RoomID string `json:"room_id"`
	UserID string `json:"user_id"`
	Data   string `json:"data"`
}

type threadSafeWriter struct {
	*websocket.Conn
	sync.Mutex
}

func (t *threadSafeWriter) WriteJSON(v interface{}) error {
	t.Lock()
	defer t.Unlock()
	return t.Conn.WriteJSON(v)
}
