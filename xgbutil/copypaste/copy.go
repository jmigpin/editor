package copypaste

import (
	"bytes"
	"encoding/binary"
	"log"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/uiutil/event"
	"github.com/jmigpin/editor/xgbutil"
)

// NOTES on other applications
// chromium seems to send an abnormal number of selection requests (also target requests) just to finally settle on what it is being provided
// thunar (or the underlying framework) seems to request immediatly the selection as soon as the selection owner is set - without explicit paste

type Copy struct {
	conn  *xgb.Conn
	win   xproto.Window
	reply chan *xproto.SelectionNotifyEvent

	clipboardStr string
	primaryStr   string
}

var CopyAtoms struct {
	UTF8_STRING xproto.Atom
	XSEL_DATA   xproto.Atom
	CLIPBOARD   xproto.Atom
	TARGETS     xproto.Atom
}

func NewCopy(conn *xgb.Conn, win xproto.Window) (*Copy, error) {
	c := &Copy{conn: conn, win: win}
	if err := xgbutil.LoadAtoms(conn, &CopyAtoms); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Copy) Set(i event.CopyPasteIndex, str string) error {
	switch i {
	case event.PrimaryCPI:
		c.primaryStr = str
		return c.set(xproto.AtomPrimary)
	case event.ClipboardCPI:
		c.clipboardStr = str
		return c.set(CopyAtoms.CLIPBOARD)
	}
	panic("unhandled index")
}
func (c *Copy) set(selection xproto.Atom) error {
	cookie := xproto.SetSelectionOwnerChecked(c.conn, c.win, selection, 0)
	return cookie.Check()
}

// Another application is asking for the data
func (c *Copy) OnSelectionRequest(ev *xproto.SelectionRequestEvent) {
	switch ev.Target {
	case CopyAtoms.UTF8_STRING:
		c.transferUTF8String(ev)
	case CopyAtoms.TARGETS:
		c.transferTargets(ev)
	default:
		// debug
		s, err := xgbutil.GetAtomName(c.conn, ev.Target)
		if err != nil {
			s = err.Error()
		}
		// TODO: have msg go up as error with evreg
		log.Printf("copy: ignored selection request: asking for type %v (%v)\n", ev.Target, s)
	}
}
func (c *Copy) transferUTF8String(ev *xproto.SelectionRequestEvent) {
	var b []byte
	switch ev.Selection {
	case xproto.AtomPrimary:
		b = []byte(c.primaryStr)
	case CopyAtoms.CLIPBOARD:
		b = []byte(c.clipboardStr)
	default:
		return
	}

	// change property on the requestor
	xproto.ChangeProperty(
		c.conn,
		xproto.PropModeReplace,
		ev.Requestor, // requestor window
		ev.Property,  // property
		ev.Target,    // type
		8,            // format
		uint32(len(b)),
		b)
	// notify the server
	sne := xproto.SelectionNotifyEvent{
		Requestor: ev.Requestor,
		Selection: ev.Selection,
		Target:    ev.Target, // type
		Property:  ev.Property,
	}
	buf := sne.Bytes()
	_ = xproto.SendEvent(c.conn,
		false,
		sne.Requestor,
		xproto.EventMaskNoEvent,
		string(buf))
}
func (c *Copy) transferTargets(ev *xproto.SelectionRequestEvent) {
	targets := []xproto.Atom{CopyAtoms.UTF8_STRING}

	tbuf := new(bytes.Buffer)
	for _, t := range targets {
		binary.Write(tbuf, binary.LittleEndian, t)
	}
	b := tbuf.Bytes()

	// change property on the requestor
	xproto.ChangeProperty(
		c.conn,
		xproto.PropModeReplace,
		ev.Requestor, // requestor window
		ev.Property,  // property
		ev.Target,    // type
		32,           // format
		uint32(len(targets)),
		b)
	// notify the server
	sne := xproto.SelectionNotifyEvent{
		Requestor: ev.Requestor,
		Selection: ev.Selection,
		Target:    ev.Target, // type
		Property:  ev.Property,
	}
	buf := sne.Bytes()
	_ = xproto.SendEvent(c.conn,
		false,
		sne.Requestor,
		xproto.EventMaskNoEvent,
		string(buf))
}

// Another application now owns the selection.
func (c *Copy) OnSelectionClear(ev *xproto.SelectionClearEvent) {
	switch ev.Selection {
	case xproto.AtomPrimary:
		c.primaryStr = ""
	case CopyAtoms.CLIPBOARD:
		c.clipboardStr = ""
	}
}
