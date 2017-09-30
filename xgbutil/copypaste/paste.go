package copypaste

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/xgbutil"
	"github.com/jmigpin/editor/xgbutil/evreg"
)

type Paste struct {
	conn *xgb.Conn
	win  xproto.Window

	evReg  *evreg.Register
	events chan<- interface{}

	requests struct {
		sync.Mutex
		s []*PasteReq
	}
}

func NewPaste(conn *xgb.Conn, win xproto.Window, evReg *evreg.Register, events chan<- interface{}) (*Paste, error) {
	if err := xgbutil.LoadAtoms(conn, &PasteAtoms); err != nil {
		return nil, err
	}
	p := &Paste{
		conn:   conn,
		win:    win,
		evReg:  evReg,
		events: events,
	}

	if evReg != nil {
		evReg.Add(xproto.SelectionNotify,
			&evreg.Callback{func(ev0 interface{}) {
				ev := ev0.(xproto.SelectionNotifyEvent)
				p.OnSelectionNotify(&ev)
			}})
	}

	return p, nil
}

func (p *Paste) RequestPrimary(data interface{}) {
	p.request(xproto.AtomPrimary, data)
}
func (p *Paste) RequestClipboard(data interface{}) {
	p.request(PasteAtoms.CLIPBOARD, data)
}
func (p *Paste) request(selection xproto.Atom, data interface{}) {
	p.requests.Lock()
	defer p.requests.Unlock()

	pr := &PasteReq{data: data}
	p.requests.s = append(p.requests.s, pr)

	p.requestDataToServer(selection)

	// cleanup if no event arrives
	tick := time.Tick(500 * time.Millisecond)
	go func() {
		<-tick
		err := func() error {
			p.requests.Lock()
			defer p.requests.Unlock()
			if !pr.done {
				p.deletePasteReq(pr)
				return fmt.Errorf("paste: request timeout")
			}
			return nil
		}()
		if err != nil {
			p.events <- err
		}
	}()
}

// After requesting the data this event comes in
func (p *Paste) OnSelectionNotify(ev *xproto.SelectionNotifyEvent) {
	pr, ok := p.getRequest(ev)
	if !ok {
		return
	}
	switch ev.Property {
	case xproto.AtomNone:
		// property is none if no owner exists
	case PasteAtoms.XSEL_DATA:
		str, err := p.extractData(ev)
		if err != nil {
			p.events <- err
			return
		}
		ev := &PasteDataEvent{Str: str, Data: pr.data}
		p.events <- &evreg.EventWrap{PasteDataEventId, ev}
	}
}

func (p *Paste) getRequest(ev *xproto.SelectionNotifyEvent) (*PasteReq, bool) {
	p.requests.Lock()
	defer p.requests.Unlock()

	// not waiting for any requests
	if len(p.requests.s) == 0 {
		return nil, false
	}

	pr := p.requests.s[0]
	pr.done = true
	p.deletePasteReq(pr)

	return pr, true
}

func (p *Paste) deletePasteReq(pr *PasteReq) {
	for i, u := range p.requests.s {
		if u == pr {
			p.requests.s = append(p.requests.s[:i], p.requests.s[i+1:]...)
			break
		}
	}
}

func (p *Paste) requestDataToServer(selection xproto.Atom) {
	_ = xproto.ConvertSelection(
		p.conn,
		p.win,
		selection,
		PasteAtoms.UTF8_STRING, // target/type
		PasteAtoms.XSEL_DATA,   // property
		0)                      // xproto.Timestamp, unable to use as an id
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

type PasteReq struct {
	done bool
	data interface{} // requestor data
}

const (
	PasteDataEventId = evreg.CopyPasteEventIdStart + iota
)

type PasteDataEvent struct {
	Str  string
	Data interface{}
}
