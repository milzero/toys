package main

import (
	"flag"
	"io/ioutil"
	"net/http"
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
	sessions      = map[string]*Session{}
)

type websocketMessage struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

func main() {
	// Parse the flags passed to program
	flag.Parse()
	log.SetReportCaller(true)
	log.SetFormatter(&log.JSONFormatter{})
	log.Infof("Starting")

	indexHTML, err := ioutil.ReadFile("index.html")
	if err != nil {
		panic(err)
	}
	indexTemplate = template.Must(template.New("").Parse(string(indexHTML)))

	http.HandleFunc("/websocket", websocketHandler)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if err := indexTemplate.Execute(w, "ws://"+r.Host+"/websocket"); err != nil {
			log.Fatal(err)
		}
	})

	log.Fatal(http.ListenAndServe(*addr, nil))
}
