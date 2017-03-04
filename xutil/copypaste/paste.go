package copypaste

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/xutil/xgbutil"
)

type Paste struct {
	conn  *xgb.Conn
	win   xproto.Window
	reply struct {
		sync.Mutex
		ch chan *xproto.SelectionNotifyEvent
	}
}

var PasteAtoms struct {
	UTF8_STRING xproto.Atom
	XSEL_DATA   xproto.Atom
	CLIPBOARD   xproto.Atom
	//TARGETS     xproto.Atom
}

func NewPaste(conn *xgb.Conn, win xproto.Window) (*Paste, error) {
	p := &Paste{conn: conn, win: win}
	if err := xgbutil.LoadAtoms(conn, &PasteAtoms); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Paste) Request() (string, error) {
	// initialize reply chan (using defer to lock)
	err := func() error {
		p.reply.Lock()
		defer p.reply.Unlock()
		if p.reply.ch != nil {
			// expecting a reply already - abort
			return fmt.Errorf("already expecting a request reply")
		}
		p.reply.ch = make(chan *xproto.SelectionNotifyEvent)
		return nil
	}()
	if err != nil {
		return "", err
	}

	// not expecting a reply after leaving
	defer func() {
		p.reply.Lock()
		defer p.reply.Unlock()
		p.reply.ch = nil
	}()

	// a reply must arrive on timeout
	ctx := context.Background()
	ctx2, _ := context.WithTimeout(ctx, 50*time.Millisecond)

	p.requestData()

	select {
	case <-ctx2.Done():
		return "", ctx2.Err()
	case ev := <-p.reply.ch: // waits for OnSelectionNotify
		return p.extractData(ev)
	}
}
func (p *Paste) requestData() {
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
	propertyOk := ev.Property == PasteAtoms.XSEL_DATA
	if !propertyOk {
		return false
	}

	p.reply.Lock()
	defer p.reply.Unlock()
	if p.reply.ch != nil { // expecting reply
		p.reply.ch <- ev
		return true
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
