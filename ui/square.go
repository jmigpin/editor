package ui

import (
	"image"

	"github.com/jmigpin/editor/uiutil"
	"github.com/jmigpin/editor/xutil/keybmap"
	"github.com/jmigpin/editor/xutil/xgbutil"
)

// Used in row and column to move and close.
type Square struct {
	C             uiutil.Container
	ui            *UI
	EvReg         *xgbutil.EventRegister
	dereg         xgbutil.EventDeregister
	buttonPressed bool
	PressPointPad image.Point

	// row visual feedback (not used in column square)
	executing bool
	active    bool // received key or button inside
	dirty     bool // content changed
	cold      bool // file on disk changed
}

func NewSquare(ui *UI) *Square {
	sq := &Square{ui: ui}
	width := SquareWidth
	sq.C.Style.MainSize = &width
	sq.C.PaintFunc = sq.paint
	sq.EvReg = xgbutil.NewEventRegister()

	r1 := sq.ui.Win.EvReg.Add(keybmap.ButtonPressEventId,
		&xgbutil.ERCallback{sq.onButtonPress})
	r2 := sq.ui.Win.EvReg.Add(keybmap.ButtonReleaseEventId,
		&xgbutil.ERCallback{sq.onButtonRelease})
	r3 := sq.ui.Win.EvReg.Add(keybmap.MotionNotifyEventId,
		&xgbutil.ERCallback{sq.onMotionNotify})
	sq.dereg.Add(r1, r2, r3)

	return sq
}
func (sq *Square) Close() {
	sq.dereg.UnregisterAll()
}
func (sq *Square) paint() {
	c := &SquareColor
	if sq.dirty {
		c = &SquareDirtyColor
	}
	if sq.executing {
		c = &SquareExecutingColor
	}
	sq.ui.FillRectangle(&sq.C.Bounds, c)

	if sq.active { // top right
		r := sq.C.Bounds
		w := (r.Max.X - r.Min.X) / 2
		r.Min.X = r.Max.X - w
		r.Max.Y = r.Min.Y + w
		sq.ui.FillRectangle(&r, &SquareActiveColor)
	}
	if sq.cold { // 2nd top right
		r := sq.C.Bounds
		w := (r.Max.X - r.Min.X) / 2
		r.Min.X = r.Max.X - w
		r.Min.Y += w
		r.Max.Y = r.Min.Y + w
		sq.ui.FillRectangle(&r, &SquareColdColor)
	}
}
func (sq *Square) onButtonPress(ev0 xgbutil.EREvent) {
	ev := ev0.(*keybmap.ButtonPressEvent)
	if !ev.Point.In(sq.C.Bounds) {
		return
	}
	sq.buttonPressed = true
	u := image.Point{sq.C.Bounds.Max.X, sq.C.Bounds.Min.Y}
	sq.PressPointPad = u.Sub(*ev.Point)
}
func (sq *Square) onButtonRelease(ev0 xgbutil.EREvent) {
	if !sq.buttonPressed {
		return
	}
	sq.buttonPressed = false
	ev := ev0.(*keybmap.ButtonReleaseEvent)
	ev2 := &SquareButtonReleaseEvent{sq, ev.Button, ev.Point}
	sq.EvReg.Emit(SquareButtonReleaseEventId, ev2)
}
func (sq *Square) onMotionNotify(ev0 xgbutil.EREvent) {
	if !sq.buttonPressed {
		return
	}
	ev := ev0.(*keybmap.MotionNotifyEvent)
	ev2 := &SquareMotionNotifyEvent{sq, ev.Mods, ev.Point}
	sq.EvReg.Emit(SquareMotionNotifyEventId, ev2)

	sq.ui.RequestMotionNotify()
}
func (sq *Square) WarpPointer() {
	sa := sq.C.Bounds
	p := sa.Min.Add(image.Pt(sa.Dx()/2, sa.Dy()/2))
	sq.ui.WarpPointer(&p)
}

func (sq *Square) Executing() bool {
	return sq.executing
}
func (sq *Square) SetExecuting(v bool) {
	if sq.executing != v {
		sq.executing = v
		sq.C.NeedPaint()
	}
}
func (sq *Square) Active() bool {
	return sq.active
}
func (sq *Square) SetActive(v bool) {
	if sq.active != v {
		sq.active = v
		sq.C.NeedPaint()
	}
}
func (sq *Square) Dirty() bool {
	return sq.dirty
}
func (sq *Square) SetDirty(v bool) {
	if sq.dirty != v {
		sq.dirty = v
		sq.C.NeedPaint()
	}
}
func (sq *Square) Cold() bool {
	return sq.cold
}
func (sq *Square) SetCold(v bool) {
	if sq.cold != v {
		sq.cold = v
		sq.C.NeedPaint()
	}
}

const (
	SquareButtonReleaseEventId = iota
	SquareMotionNotifyEventId
)

type SquareButtonReleaseEvent struct {
	Square *Square
	Button *keybmap.Button
	Point  *image.Point
}
type SquareMotionNotifyEvent struct {
	Square *Square
	Mods   keybmap.Modifiers
	Point  *image.Point
}
