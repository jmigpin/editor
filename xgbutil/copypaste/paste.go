package copypaste

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/uiutil/event"
	"github.com/jmigpin/editor/xgbutil"
	"github.com/jmigpin/editor/xgbutil/evreg"
)

type Paste struct {
	conn *xgb.Conn
	win  xproto.Window

	evReg *evreg.Register

	reply chan *xproto.SelectionNotifyEvent
	reqs  struct {
		sync.Mutex
		waiting int
	}
}

func NewPaste(conn *xgb.Conn, win xproto.Window, evReg *evreg.Register) (*Paste, error) {
	if err := xgbutil.LoadAtoms(conn, &PasteAtoms); err != nil {
		return nil, err
	}
	p := &Paste{
		conn:  conn,
		win:   win,
		evReg: evReg,
	}

	// need buffer size 1 or it can deadlock on onselectionnotify
	p.reply = make(chan *xproto.SelectionNotifyEvent, 1)

	if evReg != nil {
		evReg.Add(xproto.SelectionNotify, func(ev0 interface{}) {
			ev := ev0.(xproto.SelectionNotifyEvent)
			p.OnSelectionNotify(&ev)
		})
	}

	return p, nil
}

func (p *Paste) Get(i event.CopyPasteIndex) (string, error) {
	switch i {
	case event.PrimaryCPI:
		return p.request(xproto.AtomPrimary)
	case event.ClipboardCPI:
		return p.request(PasteAtoms.CLIPBOARD)
	default:
		return "", fmt.Errorf("unhandled index")
	}
}

func (p *Paste) request(selection xproto.Atom) (string, error) {
	p.reqs.Lock()
	p.reqs.waiting++
	p.reqs.Unlock()

	p.requestDataToServer(selection)

	tick := time.Tick(500 * time.Millisecond)
	select {

	case <-tick:
		// the receiver could lock here and do a "waiting--"

		p.reqs.Lock()
		defer p.reqs.Unlock()
		if p.reqs.waiting > 0 {
			p.reqs.waiting--
		} else {
			// consume the event that just got in
			ev := <-p.reply
			return p.getData(ev)
		}

		return "", fmt.Errorf("paste: request timeout")

	case ev := <-p.reply:
		return p.getData(ev)
	}
}

func (p *Paste) getData(ev *xproto.SelectionNotifyEvent) (string, error) {
	switch ev.Property {
	case xproto.AtomNone:
		// nothing to paste (no owner exists)
		return "", nil
	case PasteAtoms.XSEL_DATA:
		return p.extractData(ev)
	default:
		return "", fmt.Errorf("unhandled property: %v", ev.Property)
	}
}

// After requesting the data this event comes in
func (p *Paste) OnSelectionNotify(ev *xproto.SelectionNotifyEvent) {
	// check if it is a paste event
	switch ev.Property {
	case xproto.AtomNone:
	case PasteAtoms.XSEL_DATA:
	default:
		return
	}

	//log.Printf("%+v", spew.Sdump(ev))

	p.reqs.Lock()
	defer p.reqs.Unlock()
	if p.reqs.waiting == 0 {
		return
	}
	p.reqs.waiting--
	p.reply <- ev
}

func (p *Paste) requestDataToServer(selection xproto.Atom) {
	_ = xproto.ConvertSelection(
		p.conn,
		p.win,
		selection,
		PasteAtoms.UTF8_STRING, // target/type
		PasteAtoms.XSEL_DATA,   // property
		xproto.TimeCurrentTime)
}

func (p *Paste) extractData(ev *xproto.SelectionNotifyEvent) (string, error) {
	if ev.Target != PasteAtoms.UTF8_STRING {
		s, err := xgbutil.GetAtomName(p.conn, ev.Target)
		if err != nil {
			s = err.Error()
		}
		return "", fmt.Errorf("paste: unexpected type: %v %v", ev.Target, s)
	}
	cookie := xproto.GetProperty(
		p.conn,
		false, // delete
		ev.Requestor,
		ev.Property,    // property that contains the data
		ev.Target,      // type
		0,              // long offset
		math.MaxUint32) // long length
	reply, err := cookie.Reply()
	if err != nil {
		return "", err
	}
	return string(reply.Value), nil
}

var PasteAtoms struct {
	UTF8_STRING xproto.Atom
	XSEL_DATA   xproto.Atom
	CLIPBOARD   xproto.Atom
	//TARGETS     xproto.Atom
}
