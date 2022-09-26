package handler

import (
	"net/http"

	"github.com/milzero/toys/service"
)

func Handler() {
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("./client"))))
	http.HandleFunc("/webrtc", service.Server.WebsocketHandler)
}
