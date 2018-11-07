package copypaste

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/driver/xgbutil"
	"github.com/jmigpin/editor/util/chanutil"
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/pkg/errors"
)

type Paste struct {
	conn *xgb.Conn
	win  xproto.Window
	sch  *chanutil.NBChan // selectionnotify
	pch  *chanutil.NBChan // propertynotify
}

func NewPaste(conn *xgb.Conn, win xproto.Window) (*Paste, error) {
	if err := xgbutil.LoadAtoms(conn, &PasteAtoms); err != nil {
		return nil, err
	}
	p := &Paste{
		conn: conn,
		win:  win,
	}
	p.sch = chanutil.NewNBChan()
	p.pch = chanutil.NewNBChan()
	return p, nil
}

//----------

func (p *Paste) Get(index event.CopyPasteIndex, fn func(string, error)) {
	go func() {
		s, err := p.get2(index)
		fn(s, err)
	}()
}

func (p *Paste) get2(index event.CopyPasteIndex) (string, error) {
	switch index {
	case event.CPIPrimary:
		return p.request(xproto.AtomPrimary)
	case event.CPIClipboard:
		return p.request(PasteAtoms.CLIPBOARD)
	default:
		return "", fmt.Errorf("unhandled index")
	}
}

//----------

func (p *Paste) request(selection xproto.Atom) (string, error) {
	// TODO: handle timestamps to force only one paste at a time?

	p.sch.NewBufChan(1)
	defer p.sch.NewBufChan(0)

	p.pch.NewBufChan(8)
	defer p.pch.NewBufChan(0)

	p.requestData(selection)

	v, err := p.sch.Receive(1000 * time.Millisecond)
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
		PasteAtoms.UTF8_STRING, // target/type
		PasteAtoms.XSEL_DATA,   // property
		xproto.TimeCurrentTime)
}

//----------

func (p *Paste) OnSelectionNotify(ev *xproto.SelectionNotifyEvent) {
	// not a a paste event
	switch ev.Property {
	case xproto.AtomNone, PasteAtoms.XSEL_DATA:
	default:
		return
	}

	err := p.sch.Send(ev)
	if err != nil {
		log.Print(errors.Wrap(err, "onselectionnotify"))
	}
}

//----------

func (p *Paste) OnPropertyNotify(ev *xproto.PropertyNotifyEvent) {
	// not a a paste event
	switch ev.Atom {
	case PasteAtoms.XSEL_DATA: // property used on requestData()
	default:
		return
	}

	//log.Printf("%#v", ev)

	err := p.pch.Send(ev)
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
	case PasteAtoms.XSEL_DATA:
		if ev.Target != PasteAtoms.UTF8_STRING {
			s, _ := xgbutil.GetAtomName(p.conn, ev.Target)
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

		if reply.Type == PasteAtoms.UTF8_STRING {
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
		if reply.Type == PasteAtoms.INCR {
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
		v, err := p.pch.Receive(1000 * time.Millisecond)
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
	UTF8_STRING xproto.Atom
	XSEL_DATA   xproto.Atom
	CLIPBOARD   xproto.Atom
	INCR        xproto.Atom
	//TARGETS     xproto.Atom
}
