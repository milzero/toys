package p2p

import (
	"github.com/milzero/toys/common"
	"github.com/sirupsen/logrus"
)

type Room struct {
	Users map[string]*User
	RoomId string
	log    *logrus.Entry
}

func NewRoom(roomId string) *Room {
	return &Room{
		Users:  map[string]*User{},
		RoomId: roomId,
		log:    common.NewLog().WithField("roomId", roomId),
	}
}