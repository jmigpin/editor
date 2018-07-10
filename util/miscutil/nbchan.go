package miscutil

import (
	"errors"
	"time"
)

type NBChan struct {
	ch chan interface{}
}

func NewNBChan() *NBChan {
	ch := &NBChan{}
	ch.ch = make(chan interface{})
	return ch
}

// Non-blocking send
func (ch *NBChan) Send(v interface{}) error {
	select {
	case ch.ch <- v:
		return nil
	default:
		return errors.New("failed to send")
	}
}

func (ch *NBChan) Receive(timeout time.Duration) (interface{}, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil, errors.New("receive timeout")
	case v := <-ch.ch:
		return v, nil
	}
}

func (ch *NBChan) NewBufChan(n int) {
	ch.ch = make(chan interface{}, n)
}
