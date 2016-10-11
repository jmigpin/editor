package xutil

import (
	"image/color"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

// https://godoc.org/github.com/BurntSushi/xgbutil/xcursor

type Cursors struct {
	conn    *xgb.Conn
	cursors []xproto.Cursor
	win     xproto.Window
}

type Cursor int

const (
	ArrowCursor Cursor = iota
	VResizeCursor
	HResizeCursor
	cursorCount
)

func NewCursors(conn *xgb.Conn, win xproto.Window) (*Cursors, error) {
	cs := &Cursors{conn: conn, win: win}
	if err := cs.init(); err != nil {
		return nil, err
	}
	return cs, nil
}
func (cs *Cursors) init() error {
	cs.cursors = make([]xproto.Cursor, cursorCount)
	// arrow
	cursor, err := cs.createCursor(132)
	if err != nil {
		return err
	}
	cs.cursors[ArrowCursor] = cursor
	// vertical resize
	cursor, err = cs.createCursor(116)
	if err != nil {
		return err
	}
	cs.cursors[VResizeCursor] = cursor
	// horizontal resize
	cursor, err = cs.createCursor(108)
	if err != nil {
		return err
	}
	cs.cursors[HResizeCursor] = cursor
	return nil
}

// https://github.com/BurntSushi/xgbutil/blob/master/xcursor/cursordef.go
func (cs *Cursors) createCursor(cursorEntry uint16) (xproto.Cursor, error) {
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
	fr, fg, fb, _ := ColorUint16s(color.Black)
	br, bg, bb, _ := ColorUint16s(color.White)

	err = xproto.CreateGlyphCursorChecked(cs.conn, cursor, fontId, fontId,
		cursorEntry, cursorEntry+1, fr, fg, fb, br, bg, bb).Check()
	if err != nil {
		return 0, err
	}
	err = xproto.CloseFontChecked(cs.conn, fontId).Check()
	if err != nil {
		return 0, err
	}
	return cursor, nil
}

func (cs *Cursors) SetCursor(cursor Cursor) {
	c := cs.cursors[cursor]
	mask := uint32(xproto.CwCursor)
	values := []uint32{uint32(c)}
	cookie := xproto.ChangeWindowAttributes(cs.conn, cs.win, mask, values)
	_ = cookie // unchecked
}
