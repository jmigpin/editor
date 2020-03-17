package mousefilter

import (
	"sync"
	"time"
)

type MoveFilter struct {
	fps      int
	emitFn   func(interface{})
	isMoveFn func(interface{}) bool

	last struct {
		sync.Mutex
		timer  *time.Timer
		sent   time.Time
		moveEv interface{}
	}
}

func NewMoveFilter(fps int, emitFn func(interface{}), isMoveFn func(interface{}) bool) *MoveFilter {
	return &MoveFilter{fps: fps, emitFn: emitFn, isMoveFn: isMoveFn}
}

func (movef *MoveFilter) Filter(ev interface{}) {
	if movef.isMoveFn(ev) {
		movef.keepMoveEv(ev)
	} else {
		movef.sendMoveEv()
		movef.emitFn(ev)
	}
}

func (movef *MoveFilter) keepMoveEv(moveEv interface{}) {
	frameDur := time.Second / time.Duration(movef.fps)
	movef.last.Lock()
	defer movef.last.Unlock()
	if movef.last.timer != nil {
		// Filter by discarding sequential old move events. Just update to send the last one received when it is time.
		movef.last.moveEv = moveEv
	} else {
		// Send event immediately if the frame duration already passed
		now := time.Now()
		if now.Sub(movef.last.sent) >= frameDur {
			movef.last.sent = now
			movef.emitFn(moveEv)
		} else {
			movef.last.moveEv = moveEv // set ev to send later
			d := frameDur - now.Sub(movef.last.sent)
			movef.last.timer = time.AfterFunc(d, movef.sendMoveEv)
		}
	}
}

func (movef *MoveFilter) sendMoveEv() {
	movef.last.Lock()
	defer movef.last.Unlock()
	if movef.last.moveEv != nil {
		movef.last.sent = time.Now()
		movef.emitFn(movef.last.moveEv)
		movef.last.moveEv = nil
	}
	if movef.last.timer != nil {
		movef.last.timer.Stop()
		movef.last.timer = nil
	}
}
