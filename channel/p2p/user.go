package p2p

import (
	"fmt"

	"github.com/milzero/toys/protocol"
	"github.com/milzero/toys/protocol/transport"
)

type User struct {
	UserId string
	RoomId string
	room   *Room
	c      *transport.ThreadSafeWriter
}

func NewUser(roomId string, userID string, room *Room, c *transport.ThreadSafeWriter) *User {
	return &User{
		UserId: userID,
		RoomId: roomId,
		room:   room,
		c:      c,
	}
}

func (u *User) pass(message *protocol.Message) error {
	if u.c == nil {
		return fmt.Errorf("transport is nil")
	}

	u.c.WriteJSON(message)
	return nil
}
