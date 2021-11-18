package main

import (
	"flag"
	"io/ioutil"
	"net/http"
	"sync"
	"text/template"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

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
	log.SetReportCaller(true)
	log.SetFormatter(&log.JSONFormatter{})
	log.Infof("Starting")

	indexHTML, err := ioutil.ReadFile("index.html")
	if err != nil {
		panic(err)
	}
	indexTemplate = template.Must(template.New("").Parse(string(indexHTML)))

	http.HandleFunc("/websocket", WebsocketHandler)

	/*
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if err := indexTemplate.Execute(w, "ws://"+r.Host+"/websocket"); err != nil {
			log.Fatal(err)
		}
	})
*/
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

	message :=  websocketMessage{}
	for {
		//l , mgs , err :=c.ReadMessage()
		//log.Infof("len: %d , msg: %s ", l , string(mgs))
		err := c.ReadJSON(&message)
		if err != nil {
			log.Errorf("read message failed , %v", err)
		}

		switch message.Event {
		case "join":
			if _ , ok :=  Rooms[message.RoomID] ; !ok {
				func() {
					roomId := message.RoomID
					room := NewRoom(roomId)
					Rooms[roomId] = room
					room.AddUser(message.UserID , c)
				}()
			}
			log.Infof("new comer enter userid:%s ")
		case "ack" , "hello":
			if room , ok :=  Rooms[message.RoomID] ; !ok {
				room.Handle(message)
			}
			log.Infof("")

		}
	}
}
