package chanutil

import (
	"errors"
	"fmt"
	"time"
)

// Non-blocking channel. Note: consider using syncutil.* instead.
type NBChan struct {
	ch        chan any
	LogString string
}

func NewNBChan() *NBChan {
	return NewNBChan2(0, "nbchan")
}
func NewNBChan2(n int, logS string) *NBChan {
	ch := &NBChan{
		ch:        make(chan any, n),
		LogString: logS,
	}
	return ch
}

//----------

// Send now if a receiver is watching, or fails (non-blocking) with error.
func (ch *NBChan) Send(v any) error {
	select {
	case ch.ch <- v:
		return nil
	default:
		return errors.New("failed to send")
	}
}

// Receives or fails after timeout.
func (ch *NBChan) Receive(timeout time.Duration) (any, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil, fmt.Errorf("%v: receive timeout", ch.LogString)
	case v := <-ch.ch:
		return v, nil
	}
}

//----------

func (ch *NBChan) NewBufChan(n int) {
	ch.ch = make(chan any, n)
}

// Setting the channel to zero allows a send to fail immediatly if there is no receiver waiting.
func (ch *NBChan) SetBufChanToZero() {
	ch.NewBufChan(0)
}
