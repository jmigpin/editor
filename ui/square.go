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
	buttonPressed bool
	PressPointPad image.Point
	Data          interface{} // external use
	executing     bool
	active        bool
	dirty         bool // buffer changed
	cold          bool // disk changed
}

func NewSquare(ui *UI) *Square {
	sq := &Square{ui: ui}
	width := SquareWidth
	sq.C.Style.MainSize = &width
	sq.C.PaintFunc = sq.paint

	sq.EvReg = xgbutil.NewEventRegister()

	fn := &xgbutil.ERCallback{sq.onButtonPress}
	sq.ui.Win.EvReg.Add(keybmap.ButtonPressEventId, fn)
	fn = &xgbutil.ERCallback{sq.onButtonRelease}
	sq.ui.Win.EvReg.Add(keybmap.ButtonReleaseEventId, fn)
	fn = &xgbutil.ERCallback{sq.onMotionNotify}
	sq.ui.Win.EvReg.Add(keybmap.MotionNotifyEventId, fn)

	return sq
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

	//if sq.active { // top right
	//r := sq.C.Bounds
	//w := (r.Max.X - r.Min.X) / 2
	//r.Min.X = r.Max.X - w
	//r.Max.Y = r.Min.Y + w
	//sq.FillRectangle(&r, &SquareActiveColor)
	//}
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
	ev2 := &SquareMotionNotifyEvent{sq, ev.Modifiers, ev.Point}
	sq.EvReg.Emit(SquareMotionNotifyEventId, ev2)

	sq.ui.RequestMotionNotify()
}

//func (sq *Square) onRootPointEvent(p *image.Point, ev Event) bool {
//switch ev0 := ev.(type) {
//case *MotionNotifyEvent:
//ev2 := &SquareRootMotionNotifyEvent{sq, ev0.Modifiers, p}
//sq.ui.PushEvent(ev2)
//sq.ui.RequestMotionNotify()
//case *ButtonReleaseEvent:
//// release callbacks
////sq.ui.Layout.OnPointEvent = nil

//// release event that started and ended in the area
//if p.In(sq.C.Bounds) {
//ev2 := &SquareButtonReleaseEvent{sq, ev0.Button, p}
//sq.ui.PushEvent(ev2)
//}

//ev2 := &SquareRootButtonReleaseEvent{sq, ev0.Button, p}
//sq.ui.PushEvent(ev2)
//}
//return true
//}
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
	Square    *Square
	Modifiers keybmap.Modifiers
	Point     *image.Point
}
