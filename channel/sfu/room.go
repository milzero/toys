package sfu

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/martian/log"
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

func (r *Room) AddUser(userId string, c *transport.ThreadSafeWriter) error {
	if _, ok := r.Users[userId]; !ok {
		user := NewUser(r.roomId, userId, r, c)
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
		log.Debugf("user %d -subscribe-> %s")
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
		user.UnPublish()
		r.log.Debug("unpublish event coming")
	case "subscribe":
		subs := []protocol.Subscribe{}
		err := json.Unmarshal([]byte(message.Data[:]), subs)
		if err != nil {
			log.Errorf("subscrible user: %s", err)
			return fmt.Errorf("subscrible user: %s", err)
		}

		for _, sub := range subs {
			sub, ok := r.Users[sub.UserId]
			if ok {
				sub.Subscribe(user)
			}
		}

		r.log.Debug("subscribe event coming")
	case "unsubscribe":

		subs := []protocol.Subscribe{}
		err := json.Unmarshal([]byte(message.Data[:]), subs)
		if err != nil {
			log.Errorf("subscrible user: %s", err)
			return fmt.Errorf("subscrible user: %s", err)
		}

		for _, sub := range subs {
			sub, ok := r.Users[sub.UserId]
			if ok {
				sub.Subscribe(user)
			}
		}
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
