package dragndrop

import (
	"context"
	"image"
	"math"
	"time"

	"github.com/BurntSushi/xgb/xproto"
)

type DropEvent struct {
	Window        xproto.Window
	Timestamp     xproto.Timestamp
	dnd           *Dnd
	positionEvent *PositionEvent
	replyCh       chan *xproto.SelectionNotifyEvent
}

func ParseDropEvent(buf []uint32, positionEvent *PositionEvent, dnd *Dnd) *DropEvent {
	return &DropEvent{
		Window: xproto.Window(buf[0]),
		// buf[1] // reserved flags
		Timestamp:     xproto.Timestamp(buf[2]),
		dnd:           dnd,
		positionEvent: positionEvent,
		replyCh:       make(chan *xproto.SelectionNotifyEvent, 5),
	}
}
func (drop *DropEvent) WindowPoint() (*image.Point, error) {
	return drop.positionEvent.WindowPoint()
}

func (drop *DropEvent) ReplyDeny() {
	action := drop.positionEvent.Action
	drop.dnd.sendFinished(drop.Window, action, false)
}
func (drop *DropEvent) ReplyAccepted() {
	action := drop.positionEvent.Action
	drop.dnd.sendFinished(drop.Window, action, true)
}
func (drop *DropEvent) RequestData(typ xproto.Atom) ([]byte, error) {
	// a reply must arrive on timeout
	ctx := context.Background()
	ctx2, _ := context.WithTimeout(ctx, 250*time.Millisecond)

	drop.requestDataToServer(typ)

	select {
	case <-ctx2.Done():
		return nil, ctx2.Err()
	case ev := <-drop.replyCh: // waits for OnSelectionNotify
		return drop.extractData(ev)
	}
}
func (drop *DropEvent) requestDataToServer(typ xproto.Atom) {
	// will get selection-notify event
	_ = xproto.ConvertSelection(
		drop.dnd.conn,
		drop.dnd.win,
		DndAtoms.XdndSelection,
		typ,
		xproto.AtomPrimary,
		drop.Timestamp)
}

// After requesting the data (requestDataToServer()) this event comes in
func (drop *DropEvent) OnSelectionNotify(ev *xproto.SelectionNotifyEvent) bool {
	if ev.Time != drop.Timestamp {
		return false
	}
	//log.Printf("ev %v, drop %v", *ev, *drop) // check fields to filter
	drop.replyCh <- ev
	return true
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
