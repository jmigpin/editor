package dragndrop

import (
	"fmt"
	"image"
	"log"
	"math"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/driver/xdriver/xutil"
	"github.com/jmigpin/editor/util/syncutil"
	"github.com/jmigpin/editor/util/uiutil/event"
)

// protocol: https://www.acc.umu.se/~vatten/XDND.html
// explanation with example: http://www.edwardrosten.com/code/dist/x_clipboard-1.1/paste.cc

// Drag and drop
type Dnd struct {
	conn *xgb.Conn
	win  xproto.Window
	data DndData
	sw   *syncutil.WaitForSet
}

func NewDnd(conn *xgb.Conn, win xproto.Window) (*Dnd, error) {
	if err := xutil.LoadAtoms(conn, &DndAtoms, false); err != nil {
		return nil, err
	}
	if err := xutil.LoadAtoms(conn, &DropTypeAtoms, false); err != nil {
		return nil, err
	}
	dnd := &Dnd{conn: conn, win: win}
	if err := dnd.setupWindowProperty(); err != nil {
		return nil, err
	}
	dnd.sw = syncutil.NewWaitForSet()
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

//----------

// Error could be nil.
func (dnd *Dnd) OnClientMessage(ev *xproto.ClientMessageEvent) (ev_ interface{}, _ error, ok bool) {
	if ev.Format != 32 {
		err := fmt.Errorf("dnd event: data format is not 32: %d", ev.Format)
		return nil, err, true
	}
	data := ev.Data.Data32
	switch ev.Type {
	case DndAtoms.XdndEnter:
		// first event to happen on a drag and drop
		dnd.onEnter(data)
	case DndAtoms.XdndPosition:
		// after the enter event, it follows many position events
		ev2, err := dnd.onPosition(data)
		return ev2, err, true
	case DndAtoms.XdndDrop:
		// drag released
		ev2, err := dnd.onDrop(data)
		return ev2, err, true
	case DndAtoms.XdndLeave:
		dnd.clearData()
	}
	return nil, nil, false
}

//----------

func (dnd *Dnd) onEnter(data []uint32) {
	dnd.data.hasEnter = true
	dnd.data.enter.win = xproto.Window(data[0])
	dnd.data.enter.moreThan3DataTypes = data[1]&1 == 1
	dnd.data.enter.types = []xproto.Atom{
		xproto.Atom(data[2]),
		xproto.Atom(data[3]),
		xproto.Atom(data[4]),
	}

	// DEBUG
	if dnd.data.enter.moreThan3DataTypes {
		log.Printf("TODO: dnd enter more than 3 data types")
		xutil.PrintAtomsNames(dnd.conn, dnd.data.enter.types...)
	}

	// translate types
	u := []event.DndType{}
	for _, t := range dnd.data.enter.types {
		switch t {
		case DropTypeAtoms.TextURLList:
			u = append(u, event.TextURLListDndT)
		}
	}
	dnd.data.enter.eventTypes = u
}

//----------

func (dnd *Dnd) onPosition(data []uint32) (ev interface{}, _ error) {
	// must have had a dnd enter event before
	if !dnd.data.hasEnter {
		return nil, fmt.Errorf("missing dnd enter event")
	}

	// position event window must be the same as the enter event
	win := xproto.Window(data[0])
	if win != dnd.data.enter.win {
		return nil, fmt.Errorf("bad dnd window: %v (expecting %v)", win, dnd.data.enter.win)
	}

	// point
	screenPoint := image.Point{int(data[2] >> 16), int(data[2] & 0xffff)}
	p, err := dnd.screenToWindowPoint(screenPoint)
	if err != nil {
		return nil, fmt.Errorf("unable to pass screen to window point: %w", err)
	}

	dnd.data.hasPosition = true
	dnd.data.position.point = p
	dnd.data.position.action = xproto.Atom(data[4])

	ev = &event.DndPosition{p, dnd.data.enter.eventTypes, dnd.positionReply}
	return ev, nil
}

func (dnd *Dnd) positionReply(action event.DndAction) {
	a := dnd.data.position.action
	accept := true
	switch action {
	case event.DndADeny:
		accept = false
	case event.DndACopy:
		a = DndAtoms.XdndActionCopy
	case event.DndAMove:
		a = DndAtoms.XdndActionMove
	case event.DndALink:
		a = DndAtoms.XdndActionLink
	case event.DndAAsk:
		a = DndAtoms.XdndActionAsk
	case event.DndAPrivate:
		a = DndAtoms.XdndActionPrivate
	default:
		log.Printf("unhandled dnd action %v", action)
	}
	dnd.sendStatus(dnd.data.enter.win, a, accept)
}

//----------

func (dnd *Dnd) onDrop(data []uint32) (ev interface{}, _ error) {
	// must have had a dnd position event before
	if !dnd.data.hasPosition {
		return nil, fmt.Errorf("missing dnd position event")
	}

	// drop event window must be the same as the enter event
	win := xproto.Window(data[0])
	if win != dnd.data.enter.win {
		return nil, fmt.Errorf("bad dnd window: %v (expecting %v)", win, dnd.data.enter.win)
	}

	dnd.data.hasDrop = true
	dnd.data.drop.timestamp = xproto.Timestamp(data[2])

	ev = &event.DndDrop{dnd.data.position.point, dnd.replyAcceptDrop, dnd.requestDropData}
	return ev, nil
}
func (dnd *Dnd) replyAcceptDrop(v bool) {
	dnd.sendFinished(dnd.data.enter.win, dnd.data.position.action, v)
	dnd.clearData()
}
func (dnd *Dnd) requestDropData(t event.DndType) ([]byte, error) {
	// translate type
	var t2 xproto.Atom
	switch t {
	case event.TextURLListDndT:
		t2 = DropTypeAtoms.TextURLList
	default:
		return nil, fmt.Errorf("unhandled type: %v", t)
	}

	dnd.sw.Start(1500 * time.Millisecond)
	dnd.requestData(t2)
	v, err := dnd.sw.WaitForSet()
	if err != nil {
		return nil, err
	}
	ev := v.(*xproto.SelectionNotifyEvent)

	return dnd.extractData(ev)
}

//----------

// Called after a request for data.
func (dnd *Dnd) OnSelectionNotify(ev *xproto.SelectionNotifyEvent) {
	if !dnd.data.hasDrop {
		return
	}
	// timestamps must match
	if ev.Time != dnd.data.drop.timestamp {
		return
	}

	err := dnd.sw.Set(ev)
	if err != nil {
		log.Print(fmt.Errorf("onselectionnotify: %w", err))
	}
}

//----------

func (dnd *Dnd) requestData(typ xproto.Atom) {
	// will get selection-notify event
	_ = xproto.ConvertSelection(
		dnd.conn,
		dnd.win,
		DndAtoms.XdndSelection,
		typ,
		xproto.AtomPrimary,
		dnd.data.drop.timestamp)
}
func (dnd *Dnd) extractData(ev *xproto.SelectionNotifyEvent) ([]byte, error) {
	cookie := xproto.GetProperty(
		dnd.conn,
		false, // delete,
		dnd.win,
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

func (dnd *Dnd) sendFinished(win xproto.Window, action xproto.Atom, accepted bool) {
	u := FinishedEvent{dnd.win, accepted, action}
	cme := &xproto.ClientMessageEvent{
		Type:   DndAtoms.XdndFinished,
		Window: win,
		Format: 32,
		Data:   xproto.ClientMessageDataUnionData32New(u.Data32()),
	}
	dnd.sendClientMessage(cme)
}

func (dnd *Dnd) sendStatus(win xproto.Window, action xproto.Atom, accept bool) {
	flags := uint32(StatusEventSendPositionsFlag)
	if accept {
		flags |= StatusEventAcceptFlag
	}
	u := StatusEvent{dnd.win, flags, action}
	cme := &xproto.ClientMessageEvent{
		Type:   DndAtoms.XdndStatus,
		Window: win,
		Format: 32,
		Data:   xproto.ClientMessageDataUnionData32New(u.Data32()),
	}
	dnd.sendClientMessage(cme)
}

//----------

func (dnd *Dnd) sendClientMessage(cme *xproto.ClientMessageEvent) {
	_ = xproto.SendEvent(
		dnd.conn,
		false, // propagate
		cme.Window,
		xproto.EventMaskNoEvent,
		string(cme.Bytes()))
}

func (dnd *Dnd) screenToWindowPoint(sp image.Point) (image.Point, error) {
	cookie := xproto.GetGeometry(dnd.conn, xproto.Drawable(dnd.win))
	geom, err := cookie.Reply()
	if err != nil {
		return image.Point{}, err
	}
	x := int(geom.X) + int(geom.BorderWidth)
	y := int(geom.Y) + int(geom.BorderWidth)
	winMin := image.Point{x, y}
	return sp.Sub(winMin), nil
}

func (dnd *Dnd) clearData() {
	dnd.data = DndData{}
}

//----------

type DndData struct {
	hasEnter    bool
	hasPosition bool
	hasDrop     bool
	enter       struct {
		win                xproto.Window
		types              []xproto.Atom
		moreThan3DataTypes bool
		eventTypes         []event.DndType
	}
	position struct {
		point  image.Point
		action xproto.Atom
	}
	drop struct {
		timestamp xproto.Timestamp
	}
}

//----------

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

//----------

var DropTypeAtoms struct {
	TextURLList xproto.Atom `loadAtoms:"text/uri-list"` // technically, a URL
}
