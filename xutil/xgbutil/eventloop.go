package xgbutil

import (
	"log"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

type EventLoop struct {
	q chan *ELQEvent
}

func NewEventLoop() *EventLoop {
	return &EventLoop{q: make(chan *ELQEvent, 5)}
}
func (el *EventLoop) Run(conn *xgb.Conn, er *EventRegister) {
	connCh := make(chan interface{}, 50)
	go func() {
		for {
			ev, xerr := conn.PollForEvent()
			if ev == nil && xerr == nil {
				connCh <- int(QueueEmptyEventId)

				ev, xerr = conn.WaitForEvent()
				if ev == nil && xerr == nil {
					connCh <- int(ConnectionClosedEventId)
					goto forEnd1
				}
			}
			if xerr != nil {
				connCh <- xerr
			} else if ev != nil {
				connCh <- ev
			}
		}
	forEnd1:
	}()

	for {
		select {
		case ev, ok := <-el.q:
			if !ok {
				goto forEnd2
			}
			er.Emit(ev.EventId, ev.Event)
		case ev, ok := <-connCh:
			if !ok {
				goto forEnd2
			}
			switch ev2 := ev.(type) {
			case xgb.Event:

				// bypass quick motionnotify events
				if len(connCh) > 1 {
					_, ok := ev2.(xproto.MotionNotifyEvent)
					if ok {
						// break select, go to next loop iteration
						break
					}
				}

				er.Emit(XgbEventId(ev2), ev2)
			case xgb.Error:
				er.Emit(XErrorEventId, ev2)
			case int:
				er.Emit(ev2, nil)
			}
		}
	}
forEnd2:
}
func (el *EventLoop) Enqueue(eid int, ev EREvent) {
	el.q <- &ELQEvent{eid, ev}
}
func (el *EventLoop) Close() {
	close(el.q)
}

type ELQEvent struct { // event loop q event
	EventId int
	Event   EREvent
}

const (
	UnknownEventId = iota + 1000 // avoid clash with xproto
	XErrorEventId
	ConnectionClosedEventId
	QueueEmptyEventId
	// others (just a note)
	// 1100+: keybmap events
	// 1200+: dragndrop events
)

func XgbEventId(ev xgb.Event) int {
	switch ev.(type) {
	case xproto.ExposeEvent:
		return xproto.Expose
	case xproto.KeyPressEvent:
		return xproto.KeyPress
	case xproto.KeyReleaseEvent:
		return xproto.KeyRelease
	case xproto.ButtonPressEvent:
		return xproto.ButtonPress
	case xproto.ButtonReleaseEvent:
		return xproto.ButtonRelease
	case xproto.MotionNotifyEvent:
		return xproto.MotionNotify
	case xproto.ClientMessageEvent:
		return xproto.ClientMessage
	case xproto.SelectionNotifyEvent:
		return xproto.SelectionNotify
	case xproto.SelectionRequestEvent:
		return xproto.SelectionRequest
	case xproto.SelectionClearEvent:
		return xproto.SelectionClear
	default:
		log.Printf("unhandled event: %#v", ev)
		return UnknownEventId
	}
}
