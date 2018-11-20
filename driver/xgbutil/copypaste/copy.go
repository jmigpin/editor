package copypaste

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/driver/xgbutil"
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/pkg/errors"
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
	if err := xgbutil.LoadAtoms(conn, &CopyAtoms, false); err != nil {
		return nil, err
	}
	return c, nil
}

//----------

func (c *Copy) Set(i event.CopyPasteIndex, str string) error {
	switch i {
	case event.CPIPrimary:
		c.primaryStr = str
		return c.set(CopyAtoms.Primary)
	case event.CPIClipboard:
		c.clipboardStr = str
		return c.set(CopyAtoms.Clipboard)
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
	//// DEBUG
	//target, _ := xgbutil.GetAtomName(c.conn, ev.Target)
	//sel, _ := xgbutil.GetAtomName(c.conn, ev.Selection)
	//prop, _ := xgbutil.GetAtomName(c.conn, ev.Property)
	//log.Printf("on selection request: %v %v %v", target, sel, prop)

	switch ev.Target {
	case CopyAtoms.Utf8String,
		CopyAtoms.Text,
		CopyAtoms.TextPlain,
		CopyAtoms.TextPlainCharsetUtf8,
		CopyAtoms.GtkTextBufferContents:
		if err := c.transferBytes(ev); err != nil {
			events <- err
		}
	case CopyAtoms.Targets:
		if err := c.transferTargets(ev); err != nil {
			events <- err
		}
	default:
		c.debugRequest(ev, events)
		// try to transfer bytes anyway
		if err := c.transferBytes(ev); err != nil {
			events <- err
		}
	}
}

func (c *Copy) debugRequest(ev *xproto.SelectionRequestEvent, events chan<- interface{}) {
	// atom name
	name, err := xgbutil.GetAtomName(c.conn, ev.Target)
	if err != nil {
		events <- errors.Wrap(err, "cpcopy selectionrequest atom name for target")
	}
	// debug
	msg := fmt.Sprintf("cpcopy: non-standard external request for type %v %q\n", ev.Target, name)
	events <- errors.New(msg)
}

//----------

func (c *Copy) transferBytes(ev *xproto.SelectionRequestEvent) error {
	var b []byte
	switch ev.Selection {
	case CopyAtoms.Primary:
		b = []byte(c.primaryStr)
	case CopyAtoms.Clipboard:
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

// testing: $ xclip -o -target TARGETS -selection primary

func (c *Copy) transferTargets(ev *xproto.SelectionRequestEvent) error {
	targets := []xproto.Atom{
		CopyAtoms.Targets,
		CopyAtoms.Utf8String,
		CopyAtoms.Text,
		CopyAtoms.TextPlain,
		CopyAtoms.TextPlainCharsetUtf8,
	}

	tbuf := new(bytes.Buffer)
	for _, t := range targets {
		binary.Write(tbuf, binary.LittleEndian, t)
	}

	// change property on the requestor
	c1 := xproto.ChangePropertyChecked(
		c.conn,
		xproto.PropModeReplace,
		ev.Requestor,   // requestor window
		ev.Property,    // property
		CopyAtoms.Atom, // (would not work in some cases with ev.Target)
		32,             // format
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
	case CopyAtoms.Primary:
		c.primaryStr = ""
	case CopyAtoms.Clipboard:
		c.clipboardStr = ""
	}
}

//----------

var CopyAtoms struct {
	Atom      xproto.Atom `loadAtoms:"ATOM"`
	Primary   xproto.Atom `loadAtoms:"PRIMARY"`
	Clipboard xproto.Atom `loadAtoms:"CLIPBOARD"`
	Targets   xproto.Atom `loadAtoms:"TARGETS"`

	Utf8String            xproto.Atom `loadAtoms:"UTF8_STRING"`
	Text                  xproto.Atom `loadAtoms:"TEXT"`
	TextPlain             xproto.Atom `loadAtoms:"text/plain"`
	TextPlainCharsetUtf8  xproto.Atom `loadAtoms:"text/plain;charset=utf-8"`
	GtkTextBufferContents xproto.Atom `loadAtoms:"GTK_TEXT_BUFFER_CONTENTS"`
}
