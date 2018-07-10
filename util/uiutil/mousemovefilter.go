package uiutil

import (
	"time"

	"github.com/jmigpin/editor/util/uiutil/event"
)

func MouseMoveFilterLoop(in <-chan interface{}, out chan<- interface{}, fps *int) {
	var lastMoveEv interface{}
	var ticker *time.Ticker
	var timeToSend <-chan time.Time
	var lastTimeSent time.Time

	// DEBUG
	//n := 0

	keepMoveEv := func(ev interface{}) {
		// DEBUG
		//n++

		frameDur := time.Second / time.Duration(*fps)
		lastMoveEv = ev
		if ticker == nil {
			// Send event immediately if the frame duration already passed
			now := time.Now()
			if now.Sub(lastTimeSent) >= frameDur {
				// DEBUG
				//n--

				lastTimeSent = now
				out <- lastMoveEv
			} else {
				d := frameDur - now.Sub(lastTimeSent)
				ticker = time.NewTicker(d)
				timeToSend = ticker.C
			}
		}
	}

	sendMoveEv := func() {
		// DEBUG
		//log.Printf("kept %d times before sending", n)
		//n = 0

		ticker.Stop()
		ticker = nil
		timeToSend = nil
		lastTimeSent = time.Now()
		out <- lastMoveEv
	}

	sendMoveEvIfKept := func() {
		if ticker != nil {
			sendMoveEv()
		}
	}

	for {
		select {
		case ev, ok := <-in:
			if !ok {
				goto forEnd
			}

			isMove := false
			if wi, ok := ev.(*event.WindowInput); ok {
				if _, ok := wi.Event.(*event.MouseMove); ok {
					isMove = true
					keepMoveEv(ev)
				}
			}
			if !isMove {
				sendMoveEvIfKept()
				out <- ev
			}
		case <-timeToSend:
			sendMoveEv()
		}
	}
forEnd:
}
