package sfu

import (
	"time"

	log "github.com/sirupsen/logrus"
)

type fn func(...interface{})

type Ticker struct {
	d      time.Duration
	cancel chan struct{}
	reset  chan struct{}
	stop chan struct{}
	fn   fn
}

func NewTicker(d time.Duration, fn fn) *Ticker {

	t := &Ticker{
		d:      d,
		cancel: make(chan struct{}),
		reset:  make(chan struct{}),
		stop:   make(chan struct{}),
		fn:     fn,
	}
	t.do()
	return t
}

func (t *Ticker) do() {
	log.Println(t.d)
	tick := time.NewTicker(t.d)
	go func() {
		for {
			select {
			case <-tick.C:
				t.fn()
			case <-t.cancel:
				log.Println("exit")
				t.reset <- struct{}{}
				return
			case <-t.stop:
				return
			}
		}
	}()

}

func (t *Ticker) ResetFn(f fn) {
	go func() {
		t.cancel <- struct{}{}
		<-t.reset
		t.fn = f
		go t.do()
	}()
}

func (t *Ticker) ResetDuration(d time.Duration) {
	go func() {
		t.cancel <- struct{}{}
		<-t.reset
		t.d = d
		go t.do()
	}()
}

func (t *Ticker) Stop() {
	go func() {
		t.stop <- struct{}{}
	}()
}
