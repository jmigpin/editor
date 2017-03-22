package xgbutil

import (
	"log"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

func EventLoop(conn *xgb.Conn, er *EventRegister, qChan chan *ELQEvent) {

	connCh := make(chan interface{})
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
			if ev != nil {
				connCh <- ev
			} else if xerr != nil {
				connCh <- xerr
			}
		}
	forEnd1:
	}()

	for {
		select {
		case ev, ok := <-qChan:
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
