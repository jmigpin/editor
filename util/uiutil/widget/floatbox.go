package widget

import (
	"image"
)

type FloatBox struct {
	ENode
	RefPoint image.Point
	content  Node
	ml       *MultiLayer
	fl       *FloatLayer
}

func NewFloatBox(ml *MultiLayer, fl *FloatLayer, content Node) *FloatBox {
	fb := &FloatBox{content: content, ml: ml, fl: fl}
	fb.Cursor = DefaultCursor
	fb.Append(content)
	fb.Marks.Add(MarkNotDraggable | MarkInBoundsHandlesEvent)
	fl.Append(fb)
	return fb
}

//----------

func (fb *FloatBox) Visible() bool {
	return !fb.Marks.HasAny(MarkForceZeroBounds)
}

func (fb *FloatBox) Hide() {
	fb.ml.BgLayer.RectNeedsPaint(fb.Bounds)
	fb.Marks.Add(MarkForceZeroBounds)
	fb.MarkNeedsLayout()
}

func (fb *FloatBox) Show() {
	fb.Marks.Remove(MarkForceZeroBounds)
	fb.MarkNeedsLayoutAndPaint()
}

func (fb *FloatBox) Toggle() {
	if !fb.Visible() {
		fb.Show()
	} else {
		fb.Hide()
	}
}

//----------

func (fb *FloatBox) Measure(hint image.Point) image.Point {
	panic("calling measure on floatbox")
}

func (fb *FloatBox) Layout() {
	// start with parent bounds to reduce to content bounds
	//b := fb.Parent.Embed().Bounds
	b := fb.Bounds

	// calc bounds attached to the reference point
	r := image.Rectangle{}
	r = r.Add(fb.RefPoint)
	m := fb.content.Measure(b.Size())
	r.Max = r.Min.Add(m)
	if r.Max.X > b.Max.X {
		diffX := r.Max.X - b.Max.X
		r = r.Sub(image.Point{diffX, 0})
	}
	r = r.Intersect(b)

	fb.content.Embed().Bounds = r
	fb.Bounds = r // reduce own bounds to contain events
}
