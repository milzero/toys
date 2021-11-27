package main

import (
	"encoding/json"
	"flag"
	"html/template"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var title = `
				┌─────────────────────────────────────────────────────────────────────┐
				│                                                                     │
				│                                                                     │
				│                                                                     │
				│    xx             x                                                 │
				│    xxx     xx      x    x xxx      xx                 xxx           │
				│    x  x   xxx     x     xxx xxx            xxxx       x        x  xx│
				│    x  xxxxx xx    x     xx    x     x     xx          xx      xx   x│
				│    x         x    x      x    x     x      xxxxxx      x      x    x│
				│    x         x    x      x    x     x           x   xxxxxxx   x   xx│
				│     x        x    x      x    x     x       xxxxx       x     x  xx │
				│                   x      x    x     x                   x     xxx   │
				│                                                         x           │
				│                                                        xx           │
				└────────────────────────────────────────────────────────x────────────┘
`

func initLog() {
	logger := &lumberjack.Logger{
		Filename:   "mini.log",
		MaxSize:    500,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}

	mw := io.MultiWriter(os.Stdout, logger)
	log.SetOutput(mw)
	//log.SetReportCaller(true)
	log.SetFormatter(&log.TextFormatter{
		ForceColors:               false,
		DisableColors:             false,
		ForceQuote:                true,
		DisableQuote:              false,
		EnvironmentOverrideColors: false,
		DisableTimestamp:          false,
		FullTimestamp:             false,
		TimestampFormat:           "",
		DisableSorting:            false,
		SortingFunc:               nil,
		DisableLevelTruncation:    false,
		PadLevelText:              false,
		QuoteEmptyFields:          false,
		FieldMap:                  nil,
		CallerPrettyfier:          nil,
	})
	log.SetLevel(log.DebugLevel)
}

var (
	addr     = flag.String("addr", ":8080", "http service address")
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	indexTemplate = &template.Template{}
	sessions      = map[string]*User{}
)

var Rooms = map[string]*Room{}

func main() {
	flag.Parse()
	initLog()

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
	c := &threadSafeWriter{unsafeConn, sync.Mutex{}}

	defer c.Close()

	for {
		message := websocketMessage{}
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

		err := c.ReadJSON(&message)
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

		log.Debug(message)
		switch message.Event {
		case "join":
			_, ok := Rooms[message.RoomID]
			if !ok {
				roomId := message.RoomID
				room := NewRoom(roomId)
				Rooms[roomId] = room
				room.AddUser(message.UserID, c)
			}
		case "publish", "unpublish", "subscribe", "unsubscribe", "exit", "candidate":
			if room, ok := Rooms[message.RoomID]; ok {
				room.Handle(&message)
			}
		case "answer":
			if room, ok := Rooms[message.RoomID]; ok {
				room.Handle(&message)
			}

			log.Infof("unkown event %v", message)
		}
	}
}
