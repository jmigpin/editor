package copypaste

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/v2/driver/xdriver/xutil"
	"github.com/jmigpin/editor/v2/util/syncutil"
	"github.com/jmigpin/editor/v2/util/uiutil/event"
)

type Paste struct {
	conn *xgb.Conn
	win  xproto.Window
	sw   *syncutil.WaitForSet // selectionnotify
	pw   *syncutil.WaitForSet // propertynotify
}

func NewPaste(conn *xgb.Conn, win xproto.Window) (*Paste, error) {
	if err := xutil.LoadAtoms(conn, &PasteAtoms, false); err != nil {
		return nil, err
	}
	p := &Paste{
		conn: conn,
		win:  win,
	}
	p.sw = syncutil.NewWaitForSet()
	p.pw = syncutil.NewWaitForSet()
	return p, nil
}

//----------

func (p *Paste) Get(index event.ClipboardIndex) (string, error) {
	switch index {
	case event.CIPrimary:
		return p.request(PasteAtoms.Primary)
	case event.CIClipboard:
		return p.request(PasteAtoms.Clipboard)
	default:
		return "", fmt.Errorf("unhandled index")
	}
}

//----------

func (p *Paste) request(selection xproto.Atom) (string, error) {
	// TODO: handle timestamps to force only one paste at a time?

	p.sw.Start(1500 * time.Millisecond)
	p.requestData(selection)
	v, err := p.sw.WaitForSet()
	if err != nil {
		return "", err
	}
	ev := v.(*xproto.SelectionNotifyEvent)

	//log.Printf("%#v", ev)

	return p.extractData(ev)
}

func (p *Paste) requestData(selection xproto.Atom) {
	_ = xproto.ConvertSelection(
		p.conn,
		p.win,
		selection,
		PasteAtoms.Utf8String, // target/type
		PasteAtoms.XSelData,   // property
		xproto.TimeCurrentTime)
}

//----------

func (p *Paste) OnSelectionNotify(ev *xproto.SelectionNotifyEvent) {
	// not a a paste event
	switch ev.Property {
	case xproto.AtomNone, PasteAtoms.XSelData:
	default:
		return
	}

	err := p.sw.Set(ev)
	if err != nil {
		log.Print(fmt.Errorf("onselectionnotify: %w", err))
	}
}

//----------

func (p *Paste) OnPropertyNotify(ev *xproto.PropertyNotifyEvent) {
	// not a a paste event
	switch ev.Atom {
	case PasteAtoms.XSelData: // property used on requestData()
	default:
		return
	}

	//log.Printf("%#v", ev)

	err := p.pw.Set(ev)
	if err != nil {
		//log.Print(errors.Wrap(err, "onpropertynotify"))
	}
}

//----------

func (p *Paste) extractData(ev *xproto.SelectionNotifyEvent) (string, error) {
	switch ev.Property {
	case xproto.AtomNone:
		// nothing to paste (no owner exists)
		return "", nil
	case PasteAtoms.XSelData:
		if ev.Target != PasteAtoms.Utf8String {
			s, _ := xutil.GetAtomName(p.conn, ev.Target)
			return "", fmt.Errorf("paste: unexpected type: %v %v", ev.Target, s)
		}
		return p.extractData3(ev)
	default:
		return "", fmt.Errorf("unhandled property: %v", ev.Property)
	}
}

func (p *Paste) extractData3(ev *xproto.SelectionNotifyEvent) (string, error) {
	w := []string{}
	incrMode := false
	for {
		cookie := xproto.GetProperty(
			p.conn,
			true, // delete
			ev.Requestor,
			ev.Property,    // property that contains the data
			ev.Target,      // type
			0,              // long offset
			math.MaxUint32) // long length
		reply, err := cookie.Reply()
		if err != nil {
			return "", err
		}

		if reply.Type == PasteAtoms.Utf8String {
			str := string(reply.Value)
			w = append(w, str)

			if incrMode {
				if reply.ValueLen == 0 {
					xproto.DeleteProperty(p.conn, ev.Requestor, ev.Property)
					break
				}
			} else {
				break
			}
		}

		// incr mode
		// https://tronche.com/gui/x/icccm/sec-2.html#s-2.7.2
		if reply.Type == PasteAtoms.Incr {
			incrMode = true
			xproto.DeleteProperty(p.conn, ev.Requestor, ev.Property)
			continue
		}
		if incrMode {
			err := p.waitForPropertyNewValue(ev)
			if err != nil {
				return "", err
			}
			continue
		}
	}

	return strings.Join(w, ""), nil
}

func (p *Paste) waitForPropertyNewValue(ev *xproto.SelectionNotifyEvent) error {
	for {
		p.pw.Start(1500 * time.Millisecond)
		v, err := p.pw.WaitForSet()
		if err != nil {
			return err
		}
		pev := v.(*xproto.PropertyNotifyEvent)
		if pev.Atom == ev.Property && pev.State == xproto.PropertyNewValue {
			return nil
		}
	}
}

//----------

var PasteAtoms struct {
	Primary   xproto.Atom `loadAtoms:"PRIMARY"`
	Clipboard xproto.Atom `loadAtoms:"CLIPBOARD"`
	XSelData  xproto.Atom `loadAtoms:"XSEL_DATA"`
	Incr      xproto.Atom `loadAtoms:"INCR"`

	Utf8String xproto.Atom `loadAtoms:"UTF8_STRING"`
}
