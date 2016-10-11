package ui

import "image"

// Mainly used for a row with textarea, although column also uses it.
type Square struct {
	Container
	PressPointPad image.Point
	Data          interface{} // external use
	executing     bool
	active        bool
	dirty         bool // buffer changed
	cold          bool // disk changed
}

func NewSquare() *Square {
	sq := &Square{}
	sq.Container.Painter = sq
	sq.Container.OnPointEvent = sq.onPointEvent
	return sq
}
func (sq *Square) CalcArea(area *image.Rectangle) {
	sq.Area = *area
}
func (sq *Square) Paint() {
	c := &SquareColor
	if sq.dirty {
		c = &SquareDirtyColor
	}
	if sq.executing {
		c = &SquareExecutingColor
	}
	sq.FillRectangle(&sq.Area, c)

	//if sq.active { // top right
	//r := sq.Area
	//w := (r.Max.X - r.Min.X) / 2
	//r.Min.X = r.Max.X - w
	//r.Max.Y = r.Min.Y + w
	//sq.FillRectangle(&r, &SquareActiveColor)
	//}
	if sq.cold { // 2nd top right
		r := sq.Area
		w := (r.Max.X - r.Min.X) / 2
		r.Min.X = r.Max.X - w
		r.Min.Y += w
		r.Max.Y = r.Min.Y + w
		sq.FillRectangle(&r, &SquareColdColor)
	}
}
func (sq *Square) onPointEvent(p *image.Point, ev Event) bool {
	switch ev0 := ev.(type) {
	case *ButtonPressEvent:
		// register for layout callbacks
		sq.UI.Layout.OnPointEvent = sq.onRootPointEvent
		// top right corner
		u := image.Point{sq.Area.Max.X, sq.Area.Min.Y}
		sq.PressPointPad = u.Sub(*p)
		_ = ev0
	}
	return true
}
func (sq *Square) onRootPointEvent(p *image.Point, ev Event) bool {
	switch ev0 := ev.(type) {
	case *MotionNotifyEvent:
		ev2 := &SquareRootMotionNotifyEvent{sq, ev0.Modifiers, p}
		sq.UI.PushEvent(ev2)
		sq.UI.RequestMotionNotify()
	case *ButtonReleaseEvent:
		// release callbacks
		sq.UI.Layout.OnPointEvent = nil

		// release event that started and ended in the area
		if p.In(sq.Area) {
			ev2 := &SquareButtonReleaseEvent{sq, ev0.Button, p}
			sq.UI.PushEvent(ev2)
		}

		ev2 := &SquareRootButtonReleaseEvent{sq, ev0.Button, p}
		sq.UI.PushEvent(ev2)
	}
	return true
}
func (sq *Square) WarpPointer() {
	sa := sq.Area
	p := sa.Min.Add(image.Pt(sa.Dx()/2, sa.Dy()/2))
	sq.UI.WarpPointer(&p)
}

func (sq *Square) Executing() bool {
	return sq.executing
}
func (sq *Square) SetExecuting(v bool) {
	if sq.executing != v {
		sq.executing = v
		sq.NeedPaint()
	}
}
func (sq *Square) Active() bool {
	return sq.active
}
func (sq *Square) SetActive(v bool) {
	if sq.active != v {
		sq.active = v
		sq.NeedPaint()
	}
}
func (sq *Square) Dirty() bool {
	return sq.dirty
}
func (sq *Square) SetDirty(v bool) {
	if sq.dirty != v {
		sq.dirty = v
		sq.NeedPaint()
	}
}
func (sq *Square) Cold() bool {
	return sq.cold
}
func (sq *Square) SetCold(v bool) {
	if sq.cold != v {
		sq.cold = v
		sq.NeedPaint()
	}
}
