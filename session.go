package main

import "C"
import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/pion/webrtc/v3"
	log "github.com/sirupsen/logrus"
)

type User struct {
	RoomId       string
	C            *threadSafeWriter
	Peer         *webrtc.PeerConnection
	RemoteTracks map[string]*webrtc.TrackLocalStaticRTP
	userID       string
	Tracks       map[string]bool
}

func NewUser(roomId string, userID string, c *threadSafeWriter) *User {
	return &User{RoomId: roomId, userID: userID, C: c}
}

func (u *User) OnICECandidate(i *webrtc.ICECandidate) {



	log.Infof("OnICECandidate emit")
	if i == nil {
		return
	}

	candidateString, err := json.Marshal(i.ToJSON())
	if err != nil {
		log.Info(err)
		return
	}

	if writeErr := u.C.WriteJSON(&websocketMessage{
		Event: "candidate",
		Data:  string(candidateString),
	}); writeErr != nil {
		log.Info(writeErr)
	}
}

func (u *User) OnIceStatusChange(p webrtc.PeerConnectionState) {
	switch p {
	case webrtc.PeerConnectionStateFailed:
		if err := u.Peer.Close(); err != nil {
			log.Info(err)
		}
	case webrtc.PeerConnectionStateClosed:
		log.Info("connection closed")
	}
}

func (u *User) Offer() {

	offer, err := u.Peer.CreateOffer(nil)
	if err != nil {
		log.Errorf("CreateOffer Panic")
	}

	if err = u.Peer.SetLocalDescription(offer); err != nil {
		log.Errorf("SetLocalDescription  %v , %s", err, u.C.RemoteAddr().String())
	}

	offerString, err := json.Marshal(offer)
	if err != nil {
		log.Errorf("Marshal  Offer Panic")
	}
	log.Infof("SetLocalDescription  Offer %s", u.C.RemoteAddr().String())

	if err = u.C.WriteJSON(&websocketMessage{
		Event: "offer",
		Data:  string(offerString),
	}); err != nil {
		log.Info("WriteJSON  Offer Panic")
	}
}

func (u *User) Handler(w http.ResponseWriter, r *http.Request) {
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

	ss := &User{
		RoomId:       unsafeConn.RemoteAddr().String(),
		C:            c,
		Peer:         peerConnection,
		RemoteTracks: map[string]*webrtc.TrackLocalStaticRTP{},
		userID:       unsafeConn.RemoteAddr().String(),
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

	peerConnection.OnICECandidate(u.OnICECandidate)

	message := &websocketMessage{}

	for _, session := range sessions {
		if session.RoomId != ss.RoomId {
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

func (u *User) OnTrack(t *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {

	trackLocal, _ := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())
	u.RemoteTracks[trackLocal.ID()] = trackLocal
	if len(u.RemoteTracks) == 2 {
		for _, session := range sessions {
			if session.RoomId != session.RoomId {
				for _, rtp := range session.RemoteTracks {
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
}
