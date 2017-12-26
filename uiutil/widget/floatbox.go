package widget

import (
	"image"
)

// Should be a child of a top layer (possibly MultiLayer). It relies on the parent bounds.
type FloatBox struct {
	EmbedNode
	RefPoint image.Point
}

func NewFloatBox(child Node) *FloatBox {
	fb := &FloatBox{}
	fb.Cursor = DefaultCursor
	fb.SetWrapper(fb)
	fb.Append(child)
	return fb
}
func (fb *FloatBox) Measure(hint image.Point) image.Point {
	panic("calling measure on floatbox")
}

func (fb *FloatBox) CalcChildsBounds() {
	// mark for paint nodes that the old bounds intersect
	fb.MarkNeedsPaint()

	// start with the parent bounds
	fbb := fb.Parent().Embed().Bounds

	child := fb.FirstChildInAll()

	// bounds bellow reference node
	b := image.Rect(fb.RefPoint.X, fb.RefPoint.Y, fb.RefPoint.X+fbb.Dx(), fb.RefPoint.Y+fbb.Dy())

	// measure child
	m := child.Measure(b.Size()).Add(b.Min)
	b2 := image.Rect(b.Min.X, b.Min.Y, m.X, m.Y)
	if b2.Max.X > fbb.Max.X {
		diff := image.Point{b2.Max.X - fbb.Max.X, 0}
		b2 = b2.Sub(diff)
	}
	b2 = b2.Intersect(fbb)
	child.Embed().Bounds = b2

	// set own bounds, same as the child
	fb.Embed().Bounds = b2

	child.CalcChildsBounds()
}

func (fb *FloatBox) OnInputEvent(ev0 interface{}, p image.Point) bool {
	// true=handled, don't let other layers get the event. This behavior can be overriden by parent nodes.
	return true
}

func (fb *FloatBox) ShowCalcMark(v bool) {
	hide := !v
	if hide {
		if !fb.Hidden() {
			fb.SetHidden(true)
			fb.MarkNeedsPaint()
		}
	} else {
		fb.SetHidden(false)
		fb.Wrapper().CalcChildsBounds()
		fb.MarkNeedsPaint()
	}
}
