package widget

import (
	"image"
)

// First child is bottom layer.
type MultiLayer struct {
	ContainerEmbedNode
}

func (ml *MultiLayer) MarkChildNeedsPaint(r *image.Rectangle) {
	ml.ContainerEmbedNode.MarkChildNeedsPaint(r)
	for _, c := range ml.Childs() {
		ml.visit(c, r)
	}
}

func (ml *MultiLayer) visit(n Node, r *image.Rectangle) {
	if n.Marks().NeedsPaint() {
		return
	}
	if !n.Bounds().Overlaps(*r) {
		return
	}
	if n.Bounds().Eq(*r) {
		n.MarkNeedsPaint() // highly recursive from here
		return
	}

	// if the childs union doesn't contain the rectangle, mark this node
	var u image.Rectangle
	for _, c := range n.Childs() {
		u = u.Union(c.Bounds())
	}
	if !r.In(u) {
		n.MarkNeedsPaint() // highly recursive from here
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
