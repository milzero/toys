package sfu

import (
	"errors"
	"github.com/google/martian/log"
	"github.com/milzero/toys/common"
	"github.com/milzero/toys/protocol"
	"github.com/milzero/toys/protocol/transport"
	"github.com/sirupsen/logrus"
)

type Room struct {
	Users  map[string]*User
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

func (r *Room) AddUser(userId string, c *transport.ThreadSafeWriter) error {
	if _, ok := r.Users[userId]; !ok {
		user := NewUser(r.RoomId, userId, r ,c)
		r.Users[userId] = user
	}
	return nil
}

func (r *Room) DeleteUser(userId string) error {
	if _, ok := r.Users[userId]; ok {
		delete(r.Users, userId)
	}
	return nil
}

func (r *Room) OnMediaReady(user *User) {
	log.Debugf("%s  ready", user.userID)
	for uid, u := range r.Users {
		if uid == user.userID {
			continue
		}
		u.Subscribe(user)
	}
}

func (r *Room) Handle(message *protocol.Message) error {
	r.log.Debugf("incoming message %v", message)
	userId := message.UserID
	user, ok := r.Users[userId]
	if !ok {
		return errors.New("have no user in room")
	}

	event := message.Event
	switch event {
	case "publish":
		user.Publish()
	case "unpublish":
		r.log.Debug("unpublish event coming")
	case "subscribe":
		r.log.Debug("subscribe event coming")
	case "unsubscribe":
		r.log.Debug("unsubscribe event coming")
	case "candidate":
		r.log.Debug("candidate event coming")
		user.Candidate(message)
	case "answer":
		r.log.Debug("answer event coming")
		user.Answer(message)
		r.OnNewUser(user)
	}

	return nil
}

func (r *Room) OnNewUser(user *User) error {
	log.Debugf("%s  ready", user.userID)
	for uid, u := range r.Users {
		if uid == user.userID {
			continue
		}
		user.Subscribe(u)
		u.Subscribe(user)
	}
	return nil
}
