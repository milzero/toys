package protocol

type Message struct {
	Event  string `json:"event"`
	RoomID string `json:"room_id"`
	UserID string `json:"user_id"`
	Data   string `json:"data"`
	SDP    SDP    `json:"sdp"`
}

type SDP struct {
	Type string `json:"type"`
	Sdp  string `json:"sdp"`
}

type Subscribe struct {
	UserId string
	Medias []int
}
