package p2p

import (
	"errors"
	"fmt"

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
func (r *Room) UserCount() int {
	return len(r.Users)
}
func (r *Room) RoomId() string {
	return r.roomId
}

func (r *Room) Handle(message *protocol.Message) error {
	r.log.Debugf("incoming message %v", message)
	userId := message.UserID
	_, ok := r.Users[userId]
	if !ok {
		return errors.New("have no user in room")
	}

	event := message.Event
	switch event {
	default:
		for otherId, other := range r.Users {
			if otherId != userId {
				other.pass(message)
			}
		}
	}

	return nil
}
func (r *Room) AddUser(userId string, c *transport.ThreadSafeWriter) error {
	_, ok := r.Users[userId]
	if ok {
		return fmt.Errorf("user %s already in toom %s", userId, r.roomId)
	}

	r.Users[userId] = &User{
		UserId: userId,
		RoomId: r.roomId,
		room:   r,
		c:      c,
	}
	return nil
}
func (r *Room) DeleteUser(userId string) error {
	_, ok := r.Users[userId]
	if !ok {
		return fmt.Errorf("user %s not in toom %s", userId, r.roomId)
	}

	delete(r.Users, userId)
	return nil
}
