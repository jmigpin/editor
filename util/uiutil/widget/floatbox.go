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
	fb.Append(child)
	return fb
}
func (fb *FloatBox) Measure(hint image.Point) image.Point {
	// TODO: review
	panic("calling measure on floatbox")
}

func (fb *FloatBox) CalcChildsBounds() {
	// mark for paint nodes that the old bounds intersect
	fb.MarkNeedsPaint()

	// start with the parent bounds
	b := fb.Parent().Embed().Bounds

	child := fb.FirstChildInAll()

	// bounds bellow reference node
	b2 := image.Rect(fb.RefPoint.X, fb.RefPoint.Y, fb.RefPoint.X+b.Dx(), fb.RefPoint.Y+b.Dy())

	// measure child
	m := child.Measure(b2.Size()).Add(b2.Min)
	b3 := image.Rect(b2.Min.X, b2.Min.Y, m.X, m.Y)
	if b3.Max.X > b.Max.X {
		diff := image.Point{b3.Max.X - b.Max.X, 0}
		b3 = b3.Sub(diff)
	}
	b3 = b3.Intersect(b)
	child.Embed().Bounds = b3

	// set own bounds, same as the child
	fb.Embed().Bounds = b3

	child.CalcChildsBounds()
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
