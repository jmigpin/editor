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

	values [5]bool

	// not used by columns
	// TODO: remove this, and make it generic, have editor handle what it means active/dirty/etc. This should not be here.
	executing bool // executing process
	active    bool // received key or button inside
	dirty     bool // content edited and not saved
	cold      bool // disk changes since last save
	notExist  bool // file not found
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
	//if sq.values[0] || sq.dirty {
	if sq.values[0] {
		c = &SquareDirtyColor
	}
	//if sq.values[1] || sq.executing {
	if sq.values[1] {
		c = &SquareExecutingColor
	}
	sq.ui.FillRectangle(&sq.C.Bounds, c)

	// mini-squares
	r := sq.C.Bounds
	w := (r.Max.X - r.Min.X) / 2
	r.Max.X = r.Min.X + w
	r.Max.Y = r.Min.Y + w

	//if sq.values[2] || sq.cold {
	if sq.values[2] {
		// rowcol(0,0)
		sq.ui.FillRectangle(&r, &SquareColdColor)
	}
	//if sq.values[3] || sq.active {
	if sq.values[3] {
		// rowcol(0,1)
		r2 := r
		r2.Min.X += w
		r2.Max.X += w
		sq.ui.FillRectangle(&r2, &SquareActiveColor)
	}
	//if sq.values[4] || sq.notExist {
	if sq.values[4] {
		// rowcol(1,1)
		r2 := r.Add(image.Point{w, w})
		sq.ui.FillRectangle(&r2, &SquareNotExistColor)
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

func (sq *Square) Value(i int) bool {
	return sq.values[i]
}
func (sq *Square) SetValue(i int, v bool) {
	if sq.values[i] != v {
		sq.values[i] = v
		sq.C.NeedPaint()
	}
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

//func (sq *Square) Dirty() bool {
//return sq.dirty
//}
//func (sq *Square) SetDirty(v bool) {
//if sq.dirty != v {
//sq.dirty = v
//sq.C.NeedPaint()
//}
//}
//func (sq *Square) Cold() bool {
//return sq.cold
//}
//func (sq *Square) SetCold(v bool) {
//if sq.cold != v {
//sq.cold = v
//sq.C.NeedPaint()
//}
//}
//func (sq *Square) IsNotExist() bool {
//return sq.notExist
//}
//func (sq *Square) SetNotExist(v bool) {
//if sq.notExist != v {
//sq.notExist = v
//sq.C.NeedPaint()
//}
//}

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
