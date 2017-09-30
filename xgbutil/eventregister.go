package xgbutil

import (
	"container/list"
	"log"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/shm"
	"github.com/BurntSushi/xgb/xproto"
)

type EventRegister struct {
	m map[int]*list.List

	UnhandledEventFunc func(ev interface{})
}

func NewEventRegister() *EventRegister {
	er := &EventRegister{m: make(map[int]*list.List)}
	return er
}
func (er *EventRegister) Add(evId int, cb *ERCallback) *ERRegist {
	l, ok := er.m[evId]
	if !ok {
		l = list.New()
		er.m[evId] = l
	}
	l.PushBack(cb)
	return &ERRegist{er, evId, cb}
}
func (er *EventRegister) Remove(evId int, cb *ERCallback) {
	l, ok := er.m[evId]
	if !ok {
		return
	}
	for e := l.Front(); e != nil; e = e.Next() {
		cb2 := e.Value.(*ERCallback)
		if cb2 == cb {
			l.Remove(e)
			if l.Len() == 0 {
				delete(er.m, evId)
			}
			break
		}
	}
}

func (er *EventRegister) Emit(evId int, ev interface{}) {
	l, ok := er.m[evId]
	if !ok {
		fn := er.UnhandledEventFunc
		if fn != nil {
			ev2 := &EREventData{evId, ev}
			fn(ev2)
		}
		return
	}
	for e := l.Front(); e != nil; e = e.Next() {
		cb := e.Value.(*ERCallback)
		cb.F(ev)
	}
}

type ERCallback struct {
	F func(interface{})
}

type ERRegist struct {
	evReg *EventRegister
	id    int
	cb    *ERCallback
}

func (reg *ERRegist) Unregister() {
	reg.evReg.Remove(reg.id, reg.cb)
}

type EventDeregister struct {
	v []*ERRegist
}

func (d *EventDeregister) Add(u ...*ERRegist) {
	d.v = append(d.v, u...)
}
func (d *EventDeregister) UnregisterAll() {
	for _, e := range d.v {
		e.Unregister()
	}
	d.v = []*ERRegist{}
}

// util to use in event channels
type EREventData struct {
	EventId int
	Event   interface{}
}

const (
	UnknownEventId = 1000 + iota // avoid clash with xproto
	NoOpEventId
	XErrorEventId
	ConnectionClosedEventId
	ShmCompletionEventId
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
	case shm.CompletionEvent:
		return ShmCompletionEventId
	default:
		log.Printf("unhandled event: %#v", ev)
		return UnknownEventId
	}
}
