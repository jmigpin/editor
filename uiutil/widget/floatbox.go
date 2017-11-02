package widget

import (
	"image"
)

// Should be a child of a top layer (possibly MultiLayer). It relies on the parent bounds.
type FloatBox struct {
	ShellEmbedNode
	RefPoint image.Point
}

func (fb *FloatBox) Init(child Node) {
	*fb = FloatBox{}
	fb.SetWrapper(fb)
	fb.Append(child)
}
func (fb *FloatBox) Measure(hint image.Point) image.Point {
	panic("calling measure on floatbox")
}
func (fb *FloatBox) CalcChildsBounds() {
	// start with the parent bounds
	fbb := fb.Parent().Bounds()

	child := fb.FirstChild()

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
	child.SetBounds(&b2)

	// set own bounds, same as the child
	fb.SetBounds(&b2)

	child.CalcChildsBounds()
}
