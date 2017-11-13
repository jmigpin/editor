package widget

import (
	"image"
)

// First child is bottom layer.
type MultiLayer struct {
	ContainerEmbedNode
}

func (ml *MultiLayer) OnMarkChildNeedsPaint(child Node, r *image.Rectangle) {
	// visit other layers to mark if they need paint
	for _, c := range ml.Childs() {
		if c == child {
			continue
		}
		ml.visitForNeedPaint(c, r)
	}
}

func (ml *MultiLayer) visitForNeedPaint(n Node, r *image.Rectangle) {
	ne := n.Embed()
	if ne.NeedsPaint() || ne.Hidden() || ne.NotPaintable() {
		return
	}
	if !n.Bounds().Overlaps(*r) {
		return
	}

	//log.Printf("multilayer check? %v", reflect.TypeOf(n))

	// if the childs union doesn't contain the rectangle, this node needs paint
	var u image.Rectangle
	for _, c := range n.Childs() {
		u = u.Union(c.Bounds())
	}
	if !r.In(u) {
		ne.MarkNeedsPaint() // highly recursive from here
		return
	}

	for _, c := range n.Childs() {
		ml.visitForNeedPaint(c, r)
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
