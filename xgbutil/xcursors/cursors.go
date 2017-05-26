package xcursors

import (
	"image/color"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/imageutil"
)

// https://tronche.com/gui/x/xlib/appendix/b/
// https://godoc.org/github.com/BurntSushi/xgbutil/xcursor

type Cursors struct {
	conn *xgb.Conn
	win  xproto.Window
	m    map[Cursor]xproto.Cursor
}

func NewCursors(conn *xgb.Conn, win xproto.Window) (*Cursors, error) {
	cs := &Cursors{
		conn: conn,
		win:  win,
		m:    make(map[Cursor]xproto.Cursor),
	}
	return cs, nil
}
func (cs *Cursors) SetCursor(c Cursor) error {
	xc, ok := cs.m[c]
	if !ok {
		xc2, err := cs.loadCursor(c)
		if err != nil {
			return err
		}
		cs.m[c] = xc2
		xc = xc2
	}
	mask := uint32(xproto.CwCursor)
	values := []uint32{uint32(xc)}
	_ = xproto.ChangeWindowAttributes(cs.conn, cs.win, mask, values)
	return nil
}
func (cs *Cursors) loadCursor(c Cursor) (xproto.Cursor, error) {
	return cs.loadCursor2(c, color.Black, color.White)
}
func (cs *Cursors) loadCursor2(c Cursor, fg, bg color.Color) (xproto.Cursor, error) {
	if c == XCNone {
		return 0, nil
	}
	fontId, err := xproto.NewFontId(cs.conn)
	if err != nil {
		return 0, err
	}
	cursor, err := xproto.NewCursorId(cs.conn)
	if err != nil {
		return 0, err
	}
	name := "cursor"
	err = xproto.OpenFontChecked(cs.conn, fontId, uint16(len(name)), name).Check()
	if err != nil {
		return 0, err
	}

	// colors
	ur, ug, ub, _ := imageutil.ColorUint16s(fg)
	vr, vg, vb, _ := imageutil.ColorUint16s(bg)

	err = xproto.CreateGlyphCursorChecked(
		cs.conn, cursor,
		fontId, fontId,
		uint16(c), uint16(c)+1,
		ur, ug, ub,
		vr, vg, vb).Check()
	if err != nil {
		return 0, err
	}

	err = xproto.CloseFontChecked(cs.conn, fontId).Check()
	if err != nil {
		return 0, err
	}

	return cursor, nil
}

type Cursor uint16

// Just to distinguish from the other cursors (uint16) to reset to parent window cursor. Value after last x cursor at 152.
const XCNone = 200
