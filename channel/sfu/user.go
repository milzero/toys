package sfu

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pion/rtcp"

	"github.com/google/martian/log"
	"github.com/milzero/toys/common"

	"sync"

	"github.com/milzero/toys/protocol"
	"github.com/milzero/toys/protocol/transport"

	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
)

type MediaType int

const (
	Audio MediaType = 1
	Video MediaType = 2
)

type OnMediaReady func(*User)

type User struct {
	roomId       string
	c            *transport.ThreadSafeWriter
	peer         *webrtc.PeerConnection
	remoteTracks map[string]*webrtc.TrackLocalStaticRTP
	userID       string
	publishers   map[string]*User
	log          *logrus.Entry
	room         *Room
	mtx          sync.Mutex
}

func NewUser(roomId string, userID string, room *Room, c *transport.ThreadSafeWriter) *User {

	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		logrus.Error(err)
		return nil
	}

	u := &User{roomId: roomId, userID: userID, c: c,
		peer:         peerConnection,
		remoteTracks: map[string]*webrtc.TrackLocalStaticRTP{},
		publishers:   map[string]*User{},
		room:         room,
		log:          common.NewLog().WithField("roomId", roomId).WithField("userId", userID),
	}

	u.Init()
	return u
}

func (u *User) Init() {
	u.peer.OnICECandidate(u.OnICECandidate)
	u.peer.OnTrack(u.OnTrack)
	u.peer.OnConnectionStateChange(u.OnIceStatusChange)
}

func (u *User) OnICECandidate(iceCandidate *webrtc.ICECandidate) {
	if iceCandidate == nil {
		return
	}

	u.log.Debugf("OnICECandidate emit %+v", iceCandidate.ToJSON())
	candidateString, err := json.Marshal(iceCandidate.ToJSON())
	if err != nil {
		u.log.Error(err)
		return
	}

	if writeErr := u.c.WriteJSON(&protocol.Message{
		Event: "candidate",
		Data:  string(candidateString),
	}); writeErr != nil {
		u.log.Error(writeErr)
	}
}

func (u *User) Ready() {
	u.mtx.Lock()
	var audio, video bool
	for _, track := range u.remoteTracks {
		typ := track.Kind()
		switch typ {
		case webrtc.RTPCodecTypeVideo:
			video = true
		case webrtc.RTPCodecTypeAudio:
			audio = true
		}
	}

	if audio && video {
		u.log.Info("media ready")
		if u.room != nil {
			u.log.Info("media on ready")
			u.room.OnMediaReady(u)
		}
	}
	u.mtx.Unlock()
	u.DispatchKeyFrame()

	common.NewTicker(time.Second, func(i ...interface{}) {
		u.DispatchKeyFrame()
	})
}

func (u *User) DispatchKeyFrame() {

	for _, receiver := range u.peer.GetReceivers() {
		if receiver.Track() == nil {
			continue
		}

		_ = u.peer.WriteRTCP([]rtcp.Packet{
			&rtcp.PictureLossIndication{
				MediaSSRC: uint32(receiver.Track().SSRC()),
			},
		})
	}

}

func (u *User) OnIceStatusChange(p webrtc.PeerConnectionState) {
	switch p {
	case webrtc.PeerConnectionStateFailed:
		if err := u.peer.Close(); err != nil {
			u.log.Info(err)
		}
	case webrtc.PeerConnectionStateClosed:
		u.log.Info("connection closed")
	case webrtc.PeerConnectionStateConnected:
		u.log.Info("connection connected")
	case webrtc.PeerConnectionStateConnecting:
		u.log.Info("connection connecting")
	case webrtc.PeerConnectionStateDisconnected:
		u.log.Info("connection disconnected")
	}
}

func (u *User) Offer() error {

	offer, err := u.peer.CreateOffer(nil)
	if err != nil {
		u.log.Errorf("CreateOffer Panic")
		return fmt.Errorf("create offer failed  %s", err)
	}

	if err = u.peer.SetLocalDescription(offer); err != nil {
		u.log.Errorf("SetLocalDescription  %v , %s", err, u.c.RemoteAddr().String())
		return fmt.Errorf("SetLocalDescription  %v , %s", err, u.c.RemoteAddr().String())
	}

	offerString, err := json.Marshal(offer)
	if err != nil {
		u.log.Errorf("marshal  offer faled")
		return fmt.Errorf("marshal  offer faled")
	}
	log.Infof("SetLocalDescription  Offer %s", u.userID)

	if err = u.c.WriteJSON(&protocol.Message{
		Event: "offer",
		Data:  string(offerString),
	}); err != nil {
		u.log.Info("write offer to client failed : %s", err)
		return fmt.Errorf("write offer to clienr failed : %s", err)
	}
	return nil
}

func (u *User) Publish() {

	for _, typ := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo, webrtc.RTPCodecTypeAudio} {
		if _, err := u.peer.AddTransceiverFromKind(typ, webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		}); err != nil {
			u.log.Info(err)
			return
		}
	}

	for _, sender := range u.peer.GetSenders() {
		u.log.Info("sender is %+v", sender)
	}

	for _, receiver := range u.peer.GetReceivers() {
		u.log.Info("sender is %+v", receiver)
	}

	err := u.Offer()
	if err != nil {
		u.log.Info(err)
	}
}

func (u *User) UnPublish() {
	for _, transceiver := range u.peer.GetTransceivers() {
		transceiver.Stop()
	}
	u.Offer()
}

func (u *User) UnSubscribe(user *User) {
	for _, sender := range user.peer.GetSenders() {
		u.peer.RemoveTrack(sender)
	}
}

func (u *User) Subscribe(user *User) {

	if u == nil {
		u.log.Errorf("Subscribe , but user is nil")
		return
	}

	if _, ok := u.publishers[user.userID]; !ok {
		var tracks = 0
		for _, track := range user.remoteTracks {
			u.peer.AddTrack(track)
			tracks = tracks + 1
		}
		if tracks > 0 {
			err := u.Offer()
			if err == nil {
				u.publishers[user.userID] = user
				u.log.Debugf("%s have subscribe %s", u.userID, user.userID)
			}
		}

		u.DispatchKeyFrame()
	} else {
		u.log.Warnf("%s have subscribe %s", u.userID, user.userID)
	}

	u.DispatchKeyFrame()
}

func (u *User) Candidate(message *protocol.Message) {
	candidate := webrtc.ICECandidateInit{}
	if err := json.Unmarshal([]byte(message.Data), &candidate); err != nil {
		u.log.Errorf(" json.Unmarshal ICECandidateInit err:%+v", err)
		return
	}
	if err := u.peer.AddICECandidate(candidate); err != nil {
		u.log.Errorf(" add ICECandidateInit err:%+v", err)
		return
	}
}

func (u *User) Answer(message *protocol.Message) error {
	answer := webrtc.SessionDescription{}
	if err := json.Unmarshal([]byte(message.Data), &answer); err != nil {
		u.log.Error(err)
		return err
	}

	if err := u.peer.SetRemoteDescription(answer); err != nil {
		u.log.Error(err)
		return err
	}

	return nil
}

func (u *User) OnTrack(t *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {

	trackLocal, _ := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())
	f := func() {
		u.mtx.Lock()
		u.remoteTracks[trackLocal.ID()] = trackLocal
		u.mtx.Unlock()
	}

	f()
	u.Ready()
	buf := make([]byte, 1500)
	for {
		i, _, err := t.Read(buf)
		if err != nil {
			u.log.Warnf("write to recive failed")
			return
		}
		if _, err = trackLocal.Write(buf[:i]); err != nil {
			u.log.Warnf("write to recive failed")
			return
		}
	}
}
