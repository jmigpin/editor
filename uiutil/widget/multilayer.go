package widget

import (
	"image"
)

// First child is bottom layer.
type MultiLayer struct {
	ContainerEmbedNode
}

func (ml *MultiLayer) OnMarkChildNeedsPaint(child Node, r *image.Rectangle) {
	for _, c := range ml.Childs() {
		if c == child {
			continue
		}
		ml.visit(c, r)
	}
}
func (ml *MultiLayer) visit(n Node, r *image.Rectangle) {
	if n.Embed().NeedsPaint() {
		return
	}
	if !n.Bounds().Overlaps(*r) {
		return
	}
	if n.Bounds().Eq(*r) {
		n.Embed().MarkNeedsPaint() // highly recursive from here
		return
	}

	// overlap

	// if the childs union doesn't contain the rectangle, this node needs paint
	var u image.Rectangle
	for _, c := range n.Childs() {
		u = u.Union(c.Bounds())
	}
	if !r.In(u) {
		n.Embed().MarkNeedsPaint() // highly recursive from here
		return
	}

	// visit each child to see which ones contain or partially contain the rectangle
	for _, c := range n.Childs() {
		ml.visit(c, r)
	}
}

func (ml *MultiLayer) Measure(hint image.Point) image.Point {
	panic("calling measure on multilayer")
}
func (ml *MultiLayer) CalcChildsBounds() {
	u := ml.Bounds()
	for _, n := range ml.Childs() {
		// all childs get full bounds
		n.SetBounds(&u)

		n.CalcChildsBounds()
	}
}
