package main

type Room struct {
	Users map[string]*User
	roomId   string
}

func NewRoom(roomId string) *Room {
	return &Room{
		Users: map[string]*User{},
		roomId:   roomId,
	}
}

func (r Room) AddUser(userId string , c *threadSafeWriter) error{
	if _, ok := r.Users[userId] ;!ok {
		user := NewUser(r.roomId, userId, c)
		r.Users[userId] = user
	}
	return nil
}


func (r Room) Handle( message websocketMessage) error{

	return nil
}

