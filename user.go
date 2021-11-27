package main

import (
	"encoding/json"

	"github.com/pion/webrtc/v3"
	log "github.com/sirupsen/logrus"
)

type MediaType int

const (
	Audio MediaType = 1
	Video MediaType = 2
)

type User struct {
	roomId       string
	c            *threadSafeWriter
	peer         *webrtc.PeerConnection
	remoteTracks map[string]*webrtc.TrackLocalStaticRTP
	userID       string
	tracks       map[string]bool
}

func NewUser(roomId string, userID string, c *threadSafeWriter) *User {

	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		log.Error(err)
		return nil
	}

	u := &User{roomId: roomId, userID: userID, c: c,
		peer:         peerConnection,
		remoteTracks: map[string]*webrtc.TrackLocalStaticRTP{},
		tracks:       map[string]bool{}}

	u.Init()
	return u
}

func (u User) Init() {
	u.peer.OnICECandidate(u.OnICECandidate)
	u.peer.OnTrack(u.OnTrack)
	u.peer.OnConnectionStateChange(u.OnIceStatusChange)
}

func (u *User) OnICECandidate(i *webrtc.ICECandidate) {

	if i == nil {
		return
	}

	log.Debugf("OnICECandidate emit %v", i.ToJSON())
	candidateString, err := json.Marshal(i.ToJSON())
	if err != nil {
		log.Info(err)
		return
	}

	if writeErr := u.c.WriteJSON(&websocketMessage{
		Event: "candidate",
		Data:  string(candidateString),
	}); writeErr != nil {
		log.Info(writeErr)
	}
}

func (u *User) OnIceStatusChange(p webrtc.PeerConnectionState) {
	switch p {
	case webrtc.PeerConnectionStateFailed:
		if err := u.peer.Close(); err != nil {
			log.Info(err)
		}
	case webrtc.PeerConnectionStateClosed:
		log.Info("connection closed")
	}
}

func (u *User) Offer() {

	offer, err := u.peer.CreateOffer(nil)
	if err != nil {
		log.Errorf("CreateOffer Panic")
	}

	if err = u.peer.SetLocalDescription(offer); err != nil {
		log.Errorf("SetLocalDescription  %v , %s", err, u.c.RemoteAddr().String())
	}

	offerString, err := json.Marshal(offer)
	if err != nil {
		log.Errorf("Marshal  Offer Panic")
	}
	log.Infof("SetLocalDescription  Offer %s", u.c.RemoteAddr().String())

	if err = u.c.WriteJSON(&websocketMessage{
		Event: "offer",
		Data:  string(offerString),
	}); err != nil {
		log.Info("WriteJSON  Offer Panic")
	}

}

func (u *User) Publish() {

	for _, typ := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo, webrtc.RTPCodecTypeAudio} {
		if _, err := u.peer.AddTransceiverFromKind(typ, webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		}); err != nil {
			log.Info(err)
			return
		}
	}

	u.Offer()
}

func (u *User) UnPublish() {

}

func (u *User) UnSubscribe() {

}

func (u *User) Subscribe() {

}

func (u *User) Handler(message *websocketMessage) {

	switch message.Event {
	case "publish":
		u.Publish()
	case "unpublish":
		log.Info("unpublish event coming")
	case "subscribe":
		log.Info("subscribe event coming")
	case "unsubscribe":
		log.Info("unsubscribe event coming")

	case "candidate":
		candidate := webrtc.ICECandidateInit{}
		if err := json.Unmarshal([]byte(message.Data), &candidate); err != nil {
			log.Errorf(" json.Unmarshal ICECandidateInit err:%+v", err)
			return
		}
		if err := u.peer.AddICECandidate(candidate); err != nil {
			log.Errorf(" add ICECandidateInit err:%+v", err)
			return
		}
	case "answer":
		answer := webrtc.SessionDescription{}
		if err := json.Unmarshal([]byte(message.Data), &answer); err != nil {
			log.Panic(err)
			return
		}

		if err := u.peer.SetRemoteDescription(answer); err != nil {
			log.Println(err)
			return
		}
	}
}

func (u *User) OnTrack(t *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {

	trackLocal, _ := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())
	u.remoteTracks[trackLocal.ID()] = trackLocal
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
