package dragndrop

import (
	"fmt"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/xgbutil"
)

// protocol: https://www.acc.umu.se/~vatten/XDND.html
// explanation with example: http://www.edwardrosten.com/code/dist/x_clipboard-1.1/paste.cc

// TODO: contexts with timeouts

type Dnd struct { // drag and drop
	conn  *xgb.Conn
	win   xproto.Window
	evReg *xgbutil.EventRegister // event register support
	tmp   struct {
		enterEvent    *EnterEvent    // contains supported types
		positionEvent *PositionEvent // contains position
		dropEvent     *DropEvent     // waits for onselectionreply
	}
}

func NewDnd(conn *xgb.Conn, win xproto.Window, evReg *xgbutil.EventRegister) (*Dnd, error) {
	if err := xgbutil.LoadAtoms(conn, &DndAtoms); err != nil {
		return nil, err
	}
	if err := xgbutil.LoadAtoms(conn, &DropTypeAtoms); err != nil {
		return nil, err
	}
	dnd := &Dnd{conn: conn, win: win}
	if err := dnd.setupWindowProperty(); err != nil {
		return nil, err
	}

	dnd.evReg = evReg
	dnd.evReg.Add(xproto.ClientMessage,
		&xgbutil.ERCallback{dnd.onClientMessage})
	dnd.evReg.Add(xproto.SelectionNotify,
		&xgbutil.ERCallback{dnd.onSelectionNotify})

	return dnd, nil
}

// Allow other applications to know this program is dnd aware.
func (dnd *Dnd) setupWindowProperty() error {
	data := []byte{xproto.AtomBitmap, 0, 0, 0}
	cookie := xproto.ChangePropertyChecked(
		dnd.conn,
		xproto.PropModeAppend, // mode
		dnd.win,
		DndAtoms.XdndAware, // atom
		xproto.AtomAtom,    // type
		32,                 // format: xprop says that it should be 32 bit
		uint32(len(data))/4,
		data)
	return cookie.Check()
}

func (dnd *Dnd) ClearTmp() {
	dnd.tmp.enterEvent = nil
	dnd.tmp.positionEvent = nil
	dnd.tmp.dropEvent = nil
}

func (dnd *Dnd) onClientMessage(ev0 xgbutil.EREvent) {
	ev := ev0.(xproto.ClientMessageEvent)
	err := dnd.onClientMessage2(&ev)
	if err != nil {
		dnd.evReg.Emit(ErrorEventId, err)
		return
	}
}
func (dnd *Dnd) onClientMessage2(ev *xproto.ClientMessageEvent) error {
	if ev.Format != 32 {
		return fmt.Errorf("dnd event: data format is not 32: %d", ev.Format)
	}
	data := ev.Data.Data32
	switch ev.Type {
	case DndAtoms.XdndEnter:
		// first event to happen on a drag and drop
		dnd.onEnter(data)
	case DndAtoms.XdndPosition:
		// after the enter event, it follows many position events
		ev2, err := dnd.onPosition(data)
		if err != nil {
			return err
		}
		dnd.evReg.Emit(PositionEventId, ev2)
	case DndAtoms.XdndDrop:
		// drag released
		ev2, err := dnd.onDrop(data)
		if err != nil {
			return err
		}
		dnd.evReg.Emit(DropEventId, ev2)
	case DndAtoms.XdndLeave:
		dnd.ClearTmp()
	}
	return nil
}

func (dnd *Dnd) onEnter(data []uint32) {
	ev := ParseEnterEvent(data)
	dnd.tmp.enterEvent = ev // keep event for folllowing events

	if ev.MoreThan3DataTypes {
		// TODO
		fmt.Println("dnd enter event: more then 3 data types")
		xgbutil.PrintAtomsNames(dnd.conn, ev.Types)
	}
}
func (dnd *Dnd) onPosition(data []uint32) (*PositionEvent, error) {
	if dnd.tmp.enterEvent == nil {
		return nil, fmt.Errorf("missing dnd enter event")
	}
	ev := ParsePositionEvent(data, dnd.tmp.enterEvent, dnd)
	// position event window must be the same from the enter event
	if ev.Window != dnd.tmp.enterEvent.Window {
		err := fmt.Errorf("expecting dnd from window %v, got %v", dnd.tmp.enterEvent.Window, ev.Window)
		return nil, err
	}
	dnd.tmp.positionEvent = ev // keep event for folllowing events
	return ev, nil
}
func (dnd *Dnd) onDrop(data []uint32) (*DropEvent, error) {
	if dnd.tmp.positionEvent == nil {
		return nil, fmt.Errorf("missing dnd position event")
	}
	ev := ParseDropEvent(data, dnd.tmp.positionEvent, dnd)
	// drop event window must be the same from the position event
	if ev.Window != dnd.tmp.positionEvent.Window {
		err := fmt.Errorf("expecting dnd from window %v, got %v", dnd.tmp.positionEvent.Window, ev.Window)
		return nil, err
	}
	dnd.tmp.dropEvent = ev // keep event for selection notify
	return ev, nil
}

func (dnd *Dnd) getWindowGeometry() (*xproto.GetGeometryReply, error) {
	cookie := xproto.GetGeometry(dnd.conn, xproto.Drawable(dnd.win))
	return cookie.Reply()
}

func (dnd *Dnd) sendStatus(win xproto.Window, action xproto.Atom, accept bool) {
	flags := uint32(StatusEventSendPositionsFlag)
	if accept {
		flags |= StatusEventAcceptFlag
	}
	se := StatusEvent{
		Window: dnd.win,
		Flags:  flags,
		Action: action,
	}
	cme := xproto.ClientMessageEvent{
		Type:   DndAtoms.XdndStatus,
		Window: win,
		Format: 32,
		Data:   xproto.ClientMessageDataUnionData32New(se.Data32()),
	}
	dnd.sendEvent(&cme)
}
func (dnd *Dnd) sendFinished(win xproto.Window, action xproto.Atom, accepted bool) {
	u := FinishedEvent{
		Window:   dnd.win,
		Action:   action,
		Accepted: accepted,
	}
	cme := xproto.ClientMessageEvent{
		Type:   DndAtoms.XdndFinished,
		Window: win,
		Format: 32,
		Data:   xproto.ClientMessageDataUnionData32New(u.Data32()),
	}
	dnd.sendEvent(&cme)
}
func (dnd *Dnd) sendEvent(cme *xproto.ClientMessageEvent) {
	buf := cme.Bytes()
	_ = xproto.SendEvent(
		dnd.conn,
		false, // propagate
		cme.Window,
		xproto.EventMaskNoEvent,
		string(buf))
}

// Called after a request for data.
func (dnd *Dnd) onSelectionNotify(ev0 xgbutil.EREvent) {
	ev := ev0.(xproto.SelectionNotifyEvent)
	if dnd.tmp.dropEvent != nil {
		// safe to defer clear tmp variable after onselectionnotify since the dropEvent has the data
		defer dnd.ClearTmp()
		_ = dnd.tmp.dropEvent.OnSelectionNotify(&ev)
	}
}

const (
	ErrorEventId = iota + 1200
	PositionEventId
	DropEventId
)

var DndAtoms struct {
	XdndAware    xproto.Atom
	XdndEnter    xproto.Atom
	XdndLeave    xproto.Atom
	XdndPosition xproto.Atom
	XdndStatus   xproto.Atom
	XdndDrop     xproto.Atom
	XdndFinished xproto.Atom

	XdndActionCopy    xproto.Atom
	XdndActionMove    xproto.Atom
	XdndActionLink    xproto.Atom
	XdndActionAsk     xproto.Atom
	XdndActionPrivate xproto.Atom

	XdndProxy    xproto.Atom
	XdndTypeList xproto.Atom

	XdndSelection xproto.Atom
}

var DropTypeAtoms struct {
	TextURLList xproto.Atom `loadAtoms:"text/uri-list"` // technically, a URL
}
