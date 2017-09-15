package wmprotocols

import (
	"encoding/binary"
	"log"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/xgbutil"
)

// https://tronche.com/gui/x/icccm/sec-4.html#s-4.2.8.1

type WMP struct {
	conn  *xgb.Conn
	win   xproto.Window
	evReg *xgbutil.EventRegister
}

func NewWMP(conn *xgb.Conn, win xproto.Window, evReg *xgbutil.EventRegister) (*WMP, error) {
	if err := xgbutil.LoadAtoms(conn, &atoms); err != nil {
		return nil, err
	}
	wmp := &WMP{conn: conn, win: win, evReg: evReg}
	if err := wmp.setupWindowProperty(); err != nil {
		return nil, err
	}
	evReg.Add(xproto.ClientMessage,
		&xgbutil.ERCallback{wmp.onClientMessage})
	return wmp, nil
}
func (wmp *WMP) setupWindowProperty() error {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, uint32(atoms.WM_DELETE_WINDOW))
	cookie := xproto.ChangePropertyChecked(
		wmp.conn,
		xproto.PropModeAppend, // mode
		wmp.win,
		atoms.WM_PROTOCOLS, // property
		xproto.AtomAtom,    // type
		32,                 // format: xprop says that it should be 32 bit
		uint32(len(data))/4,
		data)
	return cookie.Check()
}
func (wmp *WMP) onClientMessage(ev0 interface{}) {
	ev := ev0.(xproto.ClientMessageEvent)
	if ev.Type != atoms.WM_PROTOCOLS {
		return
	}
	if ev.Format != 32 {
		log.Printf("ev format not 32: %+v", ev)
		return
	}
	data := ev.Data.Data32
	for _, e := range data {
		atom := xproto.Atom(e)
		if atom == atoms.WM_DELETE_WINDOW {
			wmp.evReg.Emit(DeleteWindowEventId, nil)
		}
	}
}

var atoms struct {
	WM_PROTOCOLS     xproto.Atom
	WM_DELETE_WINDOW xproto.Atom
}

const (
	DeleteWindowEventId = iota
)
