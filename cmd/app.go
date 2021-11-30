package main

import (
	"encoding/json"
	"flag"
	"github.com/gorilla/websocket"
	sfu2 "github.com/milzero/toys/channel/sfu"
	common "github.com/milzero/toys/common"
	"github.com/milzero/toys/protocol"
	"github.com/milzero/toys/protocol/transport"
	"net/http"
	"sync"
)

const title = `\r\n
__          __    _      _____  _______  _____ 
\ \        / /   | |    |  __ \|__   __|/ ____|
 \ \  /\  / /___ | |__  | |__) |  | |  | |     
  \ \/  \/ // _ \| '_ \ |  _  /   | |  | |     
   \  /\  /|  __/| |_) || | \ \   | |  | |____ 
    \/  \/  \___||_.__/ |_|  \_\  |_|   \_____|
                                               
`


var (
	addr     = flag.String("addr", ":8080", "http service address")
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	sessions = map[string]*sfu2.User{}
)

var Rooms = map[string]*sfu2.Room{}

var log = common.NewLog().WithField("module" , "main")

func main() {
	flag.Parse()

	log.Info(title)
	http.HandleFunc("/", WebsocketHandler)
	log.Fatal(http.ListenAndServe(*addr, nil))
}


func WebsocketHandler(w http.ResponseWriter, r *http.Request) {
	unsafeConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Info("upgrade:", err)
		return
	}

	log.Infof("new connection coming: %s", unsafeConn.RemoteAddr().String())
	c := &transport.ThreadSafeWriter{unsafeConn, sync.Mutex{}}


	var room *sfu2.Room
	var userId string

	defer func() {
		if room != nil {
			room.DeleteUser(userId)
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

		log.Debugf("read from remote: %s , type: %d , raw message %s",
			c.RemoteAddr().String(), typ, string(p[:]))

		err = json.Unmarshal(p, &message)
		if err != nil {
			log.Errorf("Unmarshal message failed %+v", err)
			return
		}

		log.Debugf("incoming message event %+v", message)


		switch message.Event {
		case "join":
			roomId := message.RoomID
			_, ok := Rooms[roomId]
			if !ok {
				room = sfu2.NewRoom(roomId)
				Rooms[roomId] = room
			}

			room = Rooms[roomId]
			room.AddUser(message.UserID, c)
			userId = message.UserID
		case "publish", "unpublish", "subscribe", "unsubscribe", "exit", "candidate","answer":
			if room, ok := Rooms[message.RoomID]; ok {
				room.Handle(&message)
			}
		}
	}
}
