package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"sync"
	"text/template"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	log "github.com/sirupsen/logrus"
)

type Session struct {
	SessionId    string
	C            *threadSafeWriter
	Peer         *webrtc.PeerConnection
	RemoteTracks map[string]*webrtc.TrackLocalStaticRTP
	ID           string
	Tracks       map[string]bool
}

func (session *Session) Offer() {

	offer, err := session.Peer.CreateOffer(nil)
	if err != nil {
		log.Errorf("CreateOffer Panic")
	}

	if err = session.Peer.SetLocalDescription(offer); err != nil {
		log.Errorf("SetLocalDescription  %v , %s", err, session.C.RemoteAddr().String())
	}

	offerString, err := json.Marshal(offer)
	if err != nil {
		log.Errorf("Marshal  Offer Panic")
	}
	log.Infof("SetLocalDescription  Offer %s", session.C.RemoteAddr().String())

	if err = session.C.WriteJSON(&websocketMessage{
		Event: "offer",
		Data:  string(offerString),
	}); err != nil {
		log.Info("WriteJSON  Offer Panic")
	}
}

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

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	unsafeConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Info("upgrade:", err)
		return
	}

	log.Infof("new connection coming: %s", unsafeConn.RemoteAddr().String())
	c := &threadSafeWriter{unsafeConn, sync.Mutex{}}
	defer c.Close()

	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		log.Error(err)
		return
	}
	defer peerConnection.Close()

	ss := &Session{
		SessionId:    unsafeConn.RemoteAddr().String(),
		C:            c,
		Peer:         peerConnection,
		RemoteTracks: map[string]*webrtc.TrackLocalStaticRTP{},
		ID:           unsafeConn.RemoteAddr().String(),
		Tracks:       map[string]bool{},
	}
	sessions[unsafeConn.RemoteAddr().String()] = ss
	for _, typ := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo, webrtc.RTPCodecTypeAudio} {
		if _, err := peerConnection.AddTransceiverFromKind(typ, webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		}); err != nil {
			log.Info(err)
			return
		}
	}

	peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {

		log.Infof("OnICECandidate emit")
		if i == nil {
			return
		}

		candidateString, err := json.Marshal(i.ToJSON())
		if err != nil {
			log.Info(err)
			return
		}

		if writeErr := c.WriteJSON(&websocketMessage{
			Event: "candidate",
			Data:  string(candidateString),
		}); writeErr != nil {
			log.Info(writeErr)
		}
	})

	peerConnection.OnConnectionStateChange(func(p webrtc.PeerConnectionState) {
		switch p {
		case webrtc.PeerConnectionStateFailed:
			if err := peerConnection.Close(); err != nil {
				log.Info(err)
			}
		case webrtc.PeerConnectionStateClosed:
			log.Info("connection closed")
		}
	})

	peerConnection.OnTrack(func(t *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {

		trackLocal, _ := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())

		ss.RemoteTracks[trackLocal.ID()] = trackLocal
		if len(ss.RemoteTracks) == 2 {
			for _, session := range sessions {
				if session.SessionId != ss.SessionId {
					for _, rtp := range ss.RemoteTracks {
						log.Infof("add AddTrack %s", session.C.RemoteAddr().String())
						session.Peer.AddTrack(rtp)
						session.Tracks[trackLocal.ID()] = true

					}
					session.Offer()
				}
			}
		}

		buf := make([]byte, 1500)
		for {
			i, _, err := t.Read(buf)
			if err != nil {
				return
			}
			if _, err = trackLocal.Write(buf[:i]); err != nil {
				return
			}
		}

	})

	message := &websocketMessage{}

	for _, session := range sessions {
		if session.SessionId != ss.SessionId {
			for _, tr := range session.RemoteTracks {
				if _, ok := ss.Tracks[tr.ID()]; !ok {
					ss.Peer.AddTrack(tr)
					ss.Tracks[tr.ID()] = true
				}
			}
		}
	}
	ss.Offer()
	for {
		_, raw, err := c.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		} else if err := json.Unmarshal(raw, &message); err != nil {
			log.Println(err)
			return
		}

		switch message.Event {
		case "candidate":
			candidate := webrtc.ICECandidateInit{}
			if err := json.Unmarshal([]byte(message.Data), &candidate); err != nil {
				log.Println(err)
				return
			}
			if err := peerConnection.AddICECandidate(candidate); err != nil {
				log.Println(err)
				return
			}
		case "answer":
			answer := webrtc.SessionDescription{}
			log.Printf("now status: %d , answer %s ", peerConnection.SignalingState(), c.Conn.RemoteAddr().String())
			if err := json.Unmarshal([]byte(message.Data), &answer); err != nil {
				log.Panic(err)
				return
			}
			log.Printf("now status: %d , %d , id: %s ", peerConnection.SignalingState(), answer.Type, c.Conn.RemoteAddr().String())

			if err := peerConnection.SetRemoteDescription(answer); err != nil {
				log.Println(err)
				return
			}
		}
	}
}

type threadSafeWriter struct {
	*websocket.Conn
	sync.Mutex
}

func (t *threadSafeWriter) WriteJSON(v interface{}) error {
	t.Lock()
	defer t.Unlock()
	return t.Conn.WriteJSON(v)
}
