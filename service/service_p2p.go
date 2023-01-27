package service

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/google/martian/log"
	"github.com/milzero/toys/channel"
	"github.com/milzero/toys/common"
	"github.com/milzero/toys/protocol"
	"github.com/milzero/toys/protocol/transport"
	"github.com/sirupsen/logrus"
)

var (
	P2PServiceInstance = &P2PService{
		Rooms:  sync.Map{},
		logger: common.NewLog(),
	}
)

type P2PService struct {
	endpoint string
	Rooms    sync.Map
	logger   *logrus.Logger
}

func (s *P2PService) WebsocketHandler(w http.ResponseWriter, r *http.Request) {
	unsafeConn, err := upgrade.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Info("upgrade:", err)
		return
	}

	s.logger.Infof("new connection coming: %s", unsafeConn.RemoteAddr().String())
	c := &transport.ThreadSafeWriter{Conn: unsafeConn}

	var room channel.Room
	var userId string

	defer func() {
		if room != nil {
			room.DeleteUser(userId)
			roomId := room.RoomId()
			if room.UserCount() == 0 {
				s.Rooms.Delete(roomId)
			}
		}
		c.Close()
	}()

	for {
		message := protocol.Message{}
		typ, p, err := c.ReadMessage()
		if err != nil {
			log.Errorf("read message failed %+v", err)
			return
		}

		s.logger.Debugf("read from remote: %s , type: %d , raw message %s",
			c.RemoteAddr().String(), typ, string(p[:]))

		err = json.Unmarshal(p, &message)
		if err != nil {
			log.Errorf("Unmarshal message failed %+v", err)
			return
		}

		s.logger.Infof("incoming message event %+v", message)

		switch message.Event {
		case "join":
			roomId := message.RoomID
			_, ok := s.Rooms.Load(roomId)
			if !ok {
				room, _ = channel.NewRoom(channel.P2P, roomId)
				s.Rooms.Store(roomId, room)
			}
			v, ok := s.Rooms.Load(roomId)
			room := v.(channel.Room)
			room.AddUser(message.UserID, c)
			userId = message.UserID
		default:
			var ok bool
			v, ok := s.Rooms.Load(message.RoomID)
			room := v.(channel.Room)
			if ok {
				room.Handle(&message)
			}
		}
	}
}
