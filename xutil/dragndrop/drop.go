package dragndrop

import (
	"context"
	"fmt"
	"image"
	"math"
	"sync"
	"time"

	"github.com/BurntSushi/xgb/xproto"
)

type DropEvent struct {
	Window    xproto.Window
	Timestamp xproto.Timestamp

	dnd           *Dnd
	Replied       bool
	positionEvent *PositionEvent

	reply struct {
		sync.Mutex
		ch chan *xproto.SelectionNotifyEvent
	}
}

func ParseDropEvent(buf []uint32, positionEvent *PositionEvent, dnd *Dnd) *DropEvent {
	return &DropEvent{
		Window: xproto.Window(buf[0]),
		// buf[1] // reserved flags
		Timestamp:     xproto.Timestamp(buf[2]),
		dnd:           dnd,
		positionEvent: positionEvent,
	}
}

func (drop *DropEvent) WindowPoint() (*image.Point, error) {
	return drop.positionEvent.WindowPoint()
}

func (drop *DropEvent) ReplyDeny() {
	drop.Replied = true
	action := drop.positionEvent.Action
	drop.dnd.sendFinished(drop.Window, action, false)
}
func (drop *DropEvent) ReplyAccepted() {
	drop.Replied = true
	action := drop.positionEvent.Action
	drop.dnd.sendFinished(drop.Window, action, true)
}

func (drop *DropEvent) RequestData(typ xproto.Atom) ([]byte, error) {
	// initialize reply chan (using defer to lock)
	err := func() error {
		drop.reply.Lock()
		defer drop.reply.Unlock()
		if drop.reply.ch != nil {
			// expecting a reply already - abort
			return fmt.Errorf("already expecting a request reply")
		}
		drop.reply.ch = make(chan *xproto.SelectionNotifyEvent)
		return nil
	}()
	if err != nil {
		return nil, err
	}

	// a reply must arrive on timeout
	ctx := context.Background()
	ctx2, _ := context.WithTimeout(ctx, 50*time.Millisecond)

	drop.requestData(typ)

	select {
	case <-ctx2.Done():
		return nil, ctx2.Err()
	case ev := <-drop.reply.ch: // waits for OnSelectionNotify
		return drop.extractData(ev)
	}
}
func (drop *DropEvent) requestData(typ xproto.Atom) {
	// will get selection-notify event
	_ = xproto.ConvertSelection(
		drop.dnd.conn,
		drop.dnd.win,
		DndAtoms.XdndSelection,
		typ,
		xproto.AtomPrimary,
		drop.Timestamp)
}

// After requesting the data (requestData()) this event comes in
func (drop *DropEvent) OnSelectionNotify(ev *xproto.SelectionNotifyEvent) bool {
	if ev.Property == xproto.AtomNone {
		return false
	}

	drop.reply.Lock()
	defer drop.reply.Unlock()
	if drop.reply.ch != nil { // expecting reply
		drop.reply.ch <- ev
		return true
	}
	return false
}
func (drop *DropEvent) extractData(ev *xproto.SelectionNotifyEvent) ([]byte, error) {
	cookie := xproto.GetProperty(
		drop.dnd.conn,
		false, // delete,
		drop.dnd.win,
		ev.Property,    // property that contains the data
		ev.Target,      // type
		0,              // long offset
		math.MaxUint32) // long length
	reply, err := cookie.Reply()
	if err != nil {
		return nil, err
	}
	return reply.Value, nil
}
