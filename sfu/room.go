package sfu

import (
	"github.com/milzero/toys/protocol"
	"github.com/milzero/toys/protocol/transport"
	log "github.com/sirupsen/logrus"
)

type Room struct {
	Users map[string]*User
	roomId   string
}

func NewRoom(roomId string) *Room {
	return &Room{
		Users: map[string]*User{},
		roomId:   roomId,
	}
}

func (r Room) AddUser(userId string , c *transport.ThreadSafeWriter) error{
	if _, ok := r.Users[userId] ;!ok {
		user := NewUser(r.roomId, userId, c)
		r.Users[userId] = user
	}
	return nil
}


func (r Room) Handle( message *protocol.Message) error{
	log.Debugf("incoming message %v" , message)
	userId := message.UserID
	user := r.Users[userId]
	user.Handler(message)
	return nil
}

