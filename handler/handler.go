package handler

import (
	"net/http"

	"github.com/milzero/toys/service"
)

func Handler() {
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("./client"))))
	http.HandleFunc("/webrtc", service.Server.WebsocketHandler)
	http.HandleFunc("/webrtc/p2p", service.P2PServiceInstance.WebsocketHandler)
	http.HandleFunc("/webrtc/mcu", service.Server.WebsocketHandler)

}
