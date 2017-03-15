package copypaste

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/xutil/xgbutil"
)

type Paste struct {
	conn            *xgb.Conn
	win             xproto.Window
	waitingForReply bool
	replyCh         chan *xproto.SelectionNotifyEvent
}

var PasteAtoms struct {
	UTF8_STRING xproto.Atom
	XSEL_DATA   xproto.Atom
	CLIPBOARD   xproto.Atom
	//TARGETS     xproto.Atom
}

func NewPaste(conn *xgb.Conn, win xproto.Window) (*Paste, error) {
	if err := xgbutil.LoadAtoms(conn, &PasteAtoms); err != nil {
		return nil, err
	}
	p := &Paste{
		conn:    conn,
		win:     win,
		replyCh: make(chan *xproto.SelectionNotifyEvent, 3),
	}
	return p, nil
}

func (p *Paste) Request() (string, error) {
	p.waitingForReply = true
	defer func() { p.waitingForReply = false }()
	// empty possible old erroneous entries (concurrency rare cases)
	for len(p.replyCh) > 0 {
		<-p.replyCh
	}
	// a reply must arrive on timeout
	ctx := context.Background()
	ctx2, _ := context.WithTimeout(ctx, 250*time.Millisecond)

	p.requestDataToServer()

	select {
	case <-ctx2.Done():
		return "", ctx2.Err()
	case ev := <-p.replyCh: // waits for OnSelectionNotify
		return p.extractData(ev)
	}
}
func (p *Paste) requestDataToServer() {
	_ = xproto.ConvertSelection(
		p.conn,
		p.win,
		//xproto.AtomPrimary, // selection // used for unix mouse selection
		PasteAtoms.CLIPBOARD,   // selection
		PasteAtoms.UTF8_STRING, // target/type
		PasteAtoms.XSEL_DATA,   // property
		0)
}

// After requesting the data (Request()) this event comes in
func (p *Paste) OnSelectionNotify(ev *xproto.SelectionNotifyEvent) bool {
	if ev.Property != PasteAtoms.XSEL_DATA {
		return false
	}
	if p.waitingForReply {
		p.replyCh <- ev
	}
	return false
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

// event register support

func (p *Paste) SetupEventRegister(evReg *xgbutil.EventRegister) {
	evReg.Add(xproto.SelectionNotify,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			ev := ev0.(xproto.SelectionNotifyEvent)
			ok := p.OnSelectionNotify(&ev)
			_ = ok
		}})
}
