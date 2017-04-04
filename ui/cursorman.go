package ui

import (
	"image"

	"github.com/jmigpin/editor/xgbutil"
	"github.com/jmigpin/editor/xgbutil/xcursors"
	"github.com/jmigpin/editor/xgbutil/xinput"
)

type CursorMan struct {
	ui          *UI
	m           map[*image.Rectangle]*CMCallback
	cursor      xcursors.Cursor
	freeCursor  xcursors.Cursor
	fixedCursor xcursors.Cursor
	fixedState  bool
}

func NewCursorMan(ui *UI) *CursorMan {
	cm := &CursorMan{
		ui: ui,
		m:  make(map[*image.Rectangle]*CMCallback),
	}

	cm.ui.Win.EvReg.Add(xinput.MotionNotifyEventId,
		&xgbutil.ERCallback{cm.onMotionNotify})

	return cm
}
func (cm *CursorMan) onMotionNotify(ev0 xgbutil.EREvent) {
	ev := ev0.(*xinput.MotionNotifyEvent)

	// always calc free cursor to have it ready when the fixed cursor gets unsed
	c := xcursors.Cursor(xcursors.XCNone)
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
func (cm *CursorMan) setCursorCached(c xcursors.Cursor) {
	if c == cm.cursor {
		return
	}
	cm.cursor = c
	cm.ui.Win.Cursors.SetCursor(c)
}
func (cm *CursorMan) SetCursor(c xcursors.Cursor) {
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
	F func(*xinput.MotionNotifyEvent) (xcursors.Cursor, bool)
}
