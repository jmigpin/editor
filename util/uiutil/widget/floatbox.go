package widget

import (
	"image"
)

// Should be a child of FloatLayer.
type FloatBox struct {
	ENode
	RefPoint image.Point
	content  Node
	ml       *MultiLayer
}

func NewFloatBox(ml *MultiLayer, content Node) *FloatBox {
	fb := &FloatBox{content: content, ml: ml}
	fb.Cursor = DefaultCursor
	fb.Append(content)
	fb.AddMarks(MarkNotDraggable | MarkInBoundsHandlesEvent)
	return fb
}

//----------

func (fb *FloatBox) Visible() bool {
	return !fb.HasAnyMarks(MarkForceZeroBounds)
}

func (fb *FloatBox) Hide() {
	fb.ml.AddMarkRect(fb.Bounds)
	fb.AddMarks(MarkForceZeroBounds)
	fb.MarkNeedsLayoutAndPaint()
}

func (fb *FloatBox) Show() {
	fb.RemoveMarks(MarkForceZeroBounds)
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
