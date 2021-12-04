package datachannel

import (
	"github.com/milzero/toys/common"
	"github.com/milzero/toys/protocol"
	"github.com/milzero/toys/protocol/transport"
	"github.com/sirupsen/logrus"
)

type Room struct {
	Users  map[string]*User
	roomId string
	log    *logrus.Entry
}

func NewRoom(roomId string) *Room {
	return &Room{
		Users:  map[string]*User{},
		roomId: roomId,
		log:    common.NewLog().WithField("roomId", roomId),
	}
}

func (r *Room) Handle(*protocol.Message) error {
	return nil
}
func (r *Room) AddUser(string, *transport.ThreadSafeWriter) error {
	return nil
}
func (r *Room) DeleteUser(string2 string) error {
	return nil
}

func (r *Room) RoomId() string {
	return ""
}

func (r *Room) UserCount() int {
	return 0
}
