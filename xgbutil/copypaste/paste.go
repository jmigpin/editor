package copypaste

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/xgbutil"
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

func NewPaste(conn *xgb.Conn, win xproto.Window, evReg *xgbutil.EventRegister) (*Paste, error) {
	if err := xgbutil.LoadAtoms(conn, &PasteAtoms); err != nil {
		return nil, err
	}
	p := &Paste{
		conn:    conn,
		win:     win,
		replyCh: make(chan *xproto.SelectionNotifyEvent, 3),
	}

	if evReg != nil {
		evReg.Add(xproto.SelectionNotify,
			&xgbutil.ERCallback{func(ev0 interface{}) {
				ev := ev0.(xproto.SelectionNotifyEvent)
				p.OnSelectionNotify(&ev)
			}})
	}

	return p, nil
}

func (p *Paste) RequestPrimary() (string, error) {
	return p.request(xproto.AtomPrimary)
}
func (p *Paste) RequestClipboard() (string, error) {
	return p.request(PasteAtoms.CLIPBOARD)
}
func (p *Paste) request(selection xproto.Atom) (string, error) {
	p.waitingForReply = true
	defer func() { p.waitingForReply = false }()
	// empty possible old erroneous entries (concurrency rare cases)
	for len(p.replyCh) > 0 {
		<-p.replyCh
	}
	// a reply must arrive on timeout
	ctx0 := context.Background()
	ctx, cancel := context.WithTimeout(ctx0, 250*time.Millisecond)
	defer cancel()

	p.requestDataToServer(selection)

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case ev := <-p.replyCh: // waits for OnSelectionNotify
		return p.extractData(ev)
	}
}
func (p *Paste) requestDataToServer(selection xproto.Atom) {
	_ = xproto.ConvertSelection(
		p.conn,
		p.win,
		selection,
		PasteAtoms.UTF8_STRING, // target/type
		PasteAtoms.XSEL_DATA,   // property
		0)
}

// After requesting the data (Request()) this event comes in
func (p *Paste) OnSelectionNotify(ev *xproto.SelectionNotifyEvent) {
	if ev.Property != PasteAtoms.XSEL_DATA {
		return
	}
	if p.waitingForReply {
		p.replyCh <- ev
	}
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
