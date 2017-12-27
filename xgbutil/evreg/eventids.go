package evreg

import (
	"log"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

const (
	UnknownEventId = 1000 + iota // avoid clash with xproto
	NoOpEventId
	//ErrorEventId
	//XErrorEventId
	//ConnectionClosedEventId
	//ShmCompletionEventId
)

// event ids for other tasks
const (
	XInputEventIdStart    = 1100
	DndEventIdStart       = 1200
	CopyPasteEventIdStart = 1250
	UIEventIdStart        = 1300
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
	//case shm.CompletionEvent:
	//return ShmCompletionEventId
	default:
		log.Printf("unhandled event: %#v", ev)
		return UnknownEventId
	}
}
