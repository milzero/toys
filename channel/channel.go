package channel

import (
	"fmt"
	"github.com/milzero/toys/channel/datachannel"
	"github.com/milzero/toys/channel/p2p"
	"github.com/milzero/toys/channel/sfu"
)

type  Type int
const (
	P2P Type = iota
	SFU
	DataChannel
)

type Room interface {
	
}

type User interface {
	
}

func NewRoom(p Type , roomId string) (Room , error) {
	switch p {
	case P2P:
		return p2p.NewRoom(roomId) , nil
	case SFU:
		return sfu.NewRoom(roomId) , nil
	case DataChannel:
		return datachannel.NewRoom(roomId) , nil
	default:
		return nil , fmt.Errorf("error room type: %d" , p)
	}
}
