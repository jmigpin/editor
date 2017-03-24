package ui

import (
	"image"

	"github.com/jmigpin/editor/xutil"
	"github.com/jmigpin/editor/xutil/keybmap"
	"github.com/jmigpin/editor/xutil/xgbutil"
)

type CursorMan struct {
	ui          *UI
	m           map[*image.Rectangle]*CMCallback
	cursor      xutil.Cursor
	freeCursor  xutil.Cursor
	fixedCursor xutil.Cursor
	fixedState  bool
}

func NewCursorMan(ui *UI) *CursorMan {
	cm := &CursorMan{
		ui: ui,
		m:  make(map[*image.Rectangle]*CMCallback),
	}

	cm.ui.Win.EvReg.Add(keybmap.MotionNotifyEventId,
		&xgbutil.ERCallback{cm.onMotionNotify})

	return cm
}
func (cm *CursorMan) onMotionNotify(ev0 xgbutil.EREvent) {
	ev := ev0.(*keybmap.MotionNotifyEvent)

	// always calc free cursor to have it ready when the fixed cursor gets unsed
	c := xutil.Cursor(xutil.XCNone)
	for r, f := range cm.m {
		if ev.Point.In(*r) {
			u, ok := f.F(ev)
			if ok {
				c = u
			}
			break
		}
	}
	cm.freeCursor = c

	c2 := cm.freeCursor
	if cm.fixedState {
		c2 = cm.fixedCursor
	}
	cm.setCursorCached(c2)
}
func (cm *CursorMan) setCursorCached(c xutil.Cursor) {
	if c == cm.cursor {
		return
	}
	cm.cursor = c
	cm.ui.Win.Cursors.SetCursor(c)
}
func (cm *CursorMan) SetCursor(c xutil.Cursor) {
	cm.fixedState = true
	cm.fixedCursor = c
	cm.setCursorCached(cm.fixedCursor)
}
func (cm *CursorMan) UnsetCursor() {
	cm.fixedState = false
	cm.setCursorCached(cm.freeCursor)
}

func (cm *CursorMan) SetBoundsCursor(r *image.Rectangle, cb *CMCallback) {
	cm.m[r] = cb
}
func (cm *CursorMan) RemoveBoundsCursor(r *image.Rectangle) {
	delete(cm.m, r)
}

type CMCallback struct {
	F func(*keybmap.MotionNotifyEvent) (xutil.Cursor, bool)
}
