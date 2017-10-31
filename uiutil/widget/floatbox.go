package widget

import (
	"image"
)

type FloatBox struct {
	ShellEmbedNode
	RefPoint image.Point
	ml       *MultiLayer
}

func (fb *FloatBox) Init(ml *MultiLayer, child Node) {
	*fb = FloatBox{ml: ml}
	fb.SetWrapper(fb)
	fb.Append(child)
}
func (fb *FloatBox) Measure(hint image.Point) image.Point {
	panic("calling measure on floatbox")
}
func (fb *FloatBox) CalcChildsBounds() {
	// start with the multilayer bounds
	fbb := fb.ml.Bounds()

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
func (fb *FloatBox) SetHidden(v bool) {
	fb.ShellEmbedNode.SetHidden(v)
	if !v {
		fb.Wrappee().wrapper.CalcChildsBounds()
	}
	// TODO: improve - lower layers need to detect what childs need paint
	// TODO: marking that all layers need paint
	fb.ml.MarkNeedsPaint()
}
func (fb *FloatBox) Paint() {
	// TODO: improve - lower layers need to detect if this layer needs paint

	// always needs paint, if hidden it won't be tested
	fb.Marks().SetNeedsPaint(true)
}
