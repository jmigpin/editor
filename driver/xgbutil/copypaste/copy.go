package copypaste

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/driver/xgbutil"
	"github.com/jmigpin/editor/util/uiutil/event"
)

type Copy struct {
	conn  *xgb.Conn
	win   xproto.Window
	reply chan *xproto.SelectionNotifyEvent

	// Data to transfer
	clipboardStr string
	primaryStr   string
}

func NewCopy(conn *xgb.Conn, win xproto.Window) (*Copy, error) {
	c := &Copy{conn: conn, win: win}
	if err := xgbutil.LoadAtoms(conn, &CopyAtoms); err != nil {
		return nil, err
	}
	return c, nil
}

//----------

func (c *Copy) Set(i event.CopyPasteIndex, str string) error {
	switch i {
	case event.CPIPrimary:
		c.primaryStr = str
		return c.set(xproto.AtomPrimary)
	case event.CPIClipboard:
		c.clipboardStr = str
		return c.set(CopyAtoms.CLIPBOARD)
	}
	panic("unhandled index")
}
func (c *Copy) set(selection xproto.Atom) error {
	cookie := xproto.SetSelectionOwnerChecked(c.conn, c.win, selection, 0)
	return cookie.Check()
}

//----------

// Another application is asking for the data
func (c *Copy) OnSelectionRequest(ev *xproto.SelectionRequestEvent, events chan<- interface{}) {
	// DEBUG
	//target, _ := xgbutil.GetAtomName(c.conn, ev.Target)
	//sel, _ := xgbutil.GetAtomName(c.conn, ev.Selection)
	//prop, _ := xgbutil.GetAtomName(c.conn, ev.Property)
	//log.Printf("on selection request: %v %v %v", target, sel, prop)

	switch ev.Target {
	default:
		// a terminal requesting 437 (text/plain;charset=utf-8) (unable to get name)
		fallthrough
	case CopyAtoms.UTF8_STRING:
		if err := c.transferBytes(ev); err != nil {
			events <- err
		}
	case CopyAtoms.TARGETS:
		if err := c.transferTargets(ev); err != nil {
			events <- err
		}
		//default:
		//	// atom name
		//	name, err := xgbutil.GetAtomName(c.conn, ev.Target)
		//	if err != nil {
		//		events <- errors.Wrap(err, "cpcopy selectionrequest atom name for target")
		//	}
		//	// debug
		//	msg := fmt.Sprintf("cpcopy: ignoring external request for type %v (%v)\n", ev.Target, name)
		//	events <- errors.New(msg)
	}
}

//----------

func (c *Copy) transferBytes(ev *xproto.SelectionRequestEvent) error {
	var b []byte
	switch ev.Selection {
	case xproto.AtomPrimary:
		b = []byte(c.primaryStr)
	case CopyAtoms.CLIPBOARD:
		b = []byte(c.clipboardStr)
	default:
		return fmt.Errorf("unhandled selection: %v", ev.Selection)
	}

	// change property on the requestor
	c1 := xproto.ChangePropertyChecked(
		c.conn,
		xproto.PropModeReplace,
		ev.Requestor, // requestor window
		ev.Property,  // property
		ev.Target,
		8, // format
		uint32(len(b)),
		b)
	if err := c1.Check(); err != nil {
		return err
	}

	// notify the server
	sne := xproto.SelectionNotifyEvent{
		Requestor: ev.Requestor,
		Selection: ev.Selection,
		Target:    ev.Target,
		Property:  ev.Property,
		Time:      ev.Time,
	}
	c2 := xproto.SendEventChecked(
		c.conn,
		false,
		sne.Requestor,
		xproto.EventMaskNoEvent,
		string(sne.Bytes()))
	return c2.Check()
}

//----------

func (c *Copy) transferTargets(ev *xproto.SelectionRequestEvent) error {
	targets := []xproto.Atom{
		CopyAtoms.UTF8_STRING,
	}

	tbuf := new(bytes.Buffer)
	for _, t := range targets {
		binary.Write(tbuf, binary.LittleEndian, t)
	}

	// change property on the requestor
	c1 := xproto.ChangePropertyChecked(
		c.conn,
		xproto.PropModeReplace,
		ev.Requestor, // requestor window
		ev.Property,  // property
		ev.Target,
		32, // format
		uint32(len(targets)),
		tbuf.Bytes())
	if err := c1.Check(); err != nil {
		return err
	}

	// notify the server
	sne := xproto.SelectionNotifyEvent{
		Requestor: ev.Requestor,
		Selection: ev.Selection,
		Target:    ev.Target,
		Property:  ev.Property,
		Time:      ev.Time,
	}
	c2 := xproto.SendEventChecked(
		c.conn,
		false,
		sne.Requestor,
		xproto.EventMaskNoEvent,
		string(sne.Bytes()))
	return c2.Check()
}

//----------

// Another application now owns the selection.
func (c *Copy) OnSelectionClear(ev *xproto.SelectionClearEvent) {
	switch ev.Selection {
	case xproto.AtomPrimary:
		c.primaryStr = ""
	case CopyAtoms.CLIPBOARD:
		c.clipboardStr = ""
	}
}

//----------

var CopyAtoms struct {
	UTF8_STRING xproto.Atom
	XSEL_DATA   xproto.Atom
	CLIPBOARD   xproto.Atom
	TARGETS     xproto.Atom
}

//const (
//	// TODO:
//	PLAIN_UTF8 xproto.Atom = 437 // (text/plain;charset=utf-8)
//)
