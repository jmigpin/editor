package wmprotocols

import (
	"encoding/binary"
	"log"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/v2/driver/xdriver/xutil"
)

// https://tronche.com/gui/x/icccm/sec-4.html#s-4.2.8.1

type WMP struct {
	conn *xgb.Conn
	win  xproto.Window
}

func NewWMP(conn *xgb.Conn, win xproto.Window) (*WMP, error) {
	if err := xutil.LoadAtoms(conn, &atoms, false); err != nil {
		return nil, err
	}
	wmp := &WMP{conn: conn, win: win}
	if err := wmp.setupWindowProperty(); err != nil {
		return nil, err
	}
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
func (wmp *WMP) OnClientMessageDeleteWindow(ev *xproto.ClientMessageEvent) (deleteWindow bool) {
	if ev.Type != atoms.WM_PROTOCOLS {
		return false
	}
	if ev.Format != 32 {
		log.Printf("ev format not 32: %+v", ev)
		return false
	}
	data := ev.Data.Data32
	for _, e := range data {
		atom := xproto.Atom(e)
		if atom == atoms.WM_DELETE_WINDOW {
			return true
		}
	}
	return false
}

var atoms struct {
	WM_PROTOCOLS     xproto.Atom
	WM_DELETE_WINDOW xproto.Atom
}
