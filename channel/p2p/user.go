package p2p

import "github.com/milzero/toys/protocol/transport"

type User struct {
	UserId string
	RoomId string
	room 	*Room
	c *transport.ThreadSafeWriter
}

func NewUser(roomId string, userID string, room *Room, c *transport.ThreadSafeWriter) *User  {
	return &User{
		UserId: userID,
		RoomId: roomId,
		room:   room,
		c:      c,
	}
}
