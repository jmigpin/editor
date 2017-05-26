package xgbutil

import (
	"log"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

type EventLoop struct {
	connQ chan interface{}
	extQ  chan *ELQEvent
}

func NewEventLoop() *EventLoop {
	return &EventLoop{
		connQ: make(chan interface{}, 50),
		extQ:  make(chan *ELQEvent, 5),
	}
}
func (el *EventLoop) Run(conn *xgb.Conn, er *EventRegister) {
	go func() {
		for {
			ev, xerr := conn.PollForEvent()
			if ev == nil && xerr == nil {
				el.connQ <- int(QueueEmptyEventId)

				ev, xerr = conn.WaitForEvent()
				if ev == nil && xerr == nil {
					el.connQ <- int(ConnectionClosedEventId)
					goto forEnd1
				}
			}
			if xerr != nil {
				el.connQ <- xerr
			} else if ev != nil {
				el.connQ <- ev
			}
		}
	forEnd1:
	}()

	for {
	selectStart1:
		select {
		case ev, ok := <-el.extQ:
			if !ok {
				goto forEnd2
			}
			er.Emit(ev.EventId, ev.Event)
		case ev, ok := <-el.connQ:
			if !ok {
				goto forEnd2
			}
			switch ev2 := ev.(type) {
			case xgb.Event:

				// bypass quick motionnotify events
				if len(el.connQ) > 1 {
					_, ok := ev2.(xproto.MotionNotifyEvent)
					if ok {
						goto selectStart1
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
func (el *EventLoop) EnqueueQEmptyEventIfConnQEmpty() {
	if len(el.connQ) == 0 {
		el.connQ <- int(QueueEmptyEventId)
	}
}

//func (el *EventLoop) Enqueue(eid int, ev EREvent) {
//el.extQ <- &ELQEvent{eid, ev}
//}

func (el *EventLoop) Close() {
	close(el.extQ)
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
	case xproto.MappingNotifyEvent:
		return xproto.MappingNotify
	default:
		log.Printf("unhandled event: %#v", ev)
		return UnknownEventId
	}
}
