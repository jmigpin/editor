package xgbutil

import (
	"log"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

const (
	UnknownEventId = iota + 1000 // avoid clash with xproto
	XErrorEventId
	ConnectionClosedEventId
	QueueEmptyEventId
	// others (just a note)
	// 1100+: keybmap events
	// 1200+: dragndrop events
)

func EventLoop(conn *xgb.Conn, er *EventRegister) {
	for {
		ev, xerr := conn.PollForEvent()
		if ev == nil && xerr == nil {
			er.Emit(QueueEmptyEventId, nil)

			ev, xerr = conn.WaitForEvent()
			if ev == nil && xerr == nil {
				er.Emit(ConnectionClosedEventId, nil)
				break
			}
		}
		if xerr != nil {
			er.Emit(XErrorEventId, xerr)
		} else {
			//log.Printf("event: %#v", ev)
			er.Emit(XgbEventId(ev), ev)
		}
	}
}
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
