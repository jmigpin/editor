package copypaste

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/driver/xgbutil"
	"github.com/jmigpin/editor/util/uiutil/event"
)

type Paste struct {
	conn  *xgb.Conn
	win   xproto.Window
	reply chan *xproto.SelectionNotifyEvent
	reqs  struct {
		sync.Mutex
		waiting int
	}
}

func NewPaste(conn *xgb.Conn, win xproto.Window) (*Paste, error) {
	if err := xgbutil.LoadAtoms(conn, &PasteAtoms); err != nil {
		return nil, err
	}
	p := &Paste{
		conn: conn,
		win:  win,
	}
	p.reply = make(chan *xproto.SelectionNotifyEvent)
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

	p.requestData(selection)

	timer := time.NewTimer(500 * time.Millisecond)
	defer timer.Stop()

	select {
	case <-timer.C:
		p.reqs.Lock()
		defer p.reqs.Unlock()
		if p.reqs.waiting == 0 {
			// an event just got in and did "waiting--"
			ev := <-p.reply
			return p.extractData(ev)
		}
		p.reqs.waiting--
		return "", fmt.Errorf("paste: request timeout")

	case ev := <-p.reply:
		return p.extractData(ev)
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

	p.reqs.Lock()
	if p.reqs.waiting > 0 {
		p.reqs.waiting--
		p.reqs.Unlock() // unlock before sending event or could be locked down
		p.reply <- ev
	} else {
		p.reqs.Unlock()
	}
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

func (p *Paste) extractData(ev *xproto.SelectionNotifyEvent) (string, error) {
	switch ev.Property {
	case xproto.AtomNone:
		// nothing to paste (no owner exists)
		return "", nil
	case PasteAtoms.XSEL_DATA:
		return p.extractData2(ev)
	default:
		return "", fmt.Errorf("unhandled property: %v", ev.Property)
	}
}
func (p *Paste) extractData2(ev *xproto.SelectionNotifyEvent) (string, error) {
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
