package transport

import (
	"sync"

	"github.com/gorilla/websocket"
)

type ThreadSafeWriter struct {
	*websocket.Conn
	sync.Mutex
}

func (t *ThreadSafeWriter) WriteJSON(v interface{}) error {
	t.Lock()
	defer t.Unlock()
	return t.Conn.WriteJSON(v)
}
