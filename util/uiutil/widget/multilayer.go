package widget

import (
	"image"
)

// First child is bottom layer.
type MultiLayer struct {
	EmbedNode
}

func (ml *MultiLayer) OnMarkChildNeedsPaint(child Node, r *image.Rectangle) {
	// visit other layers to mark if they need paint
	ml.IterChilds(func(c Node) {
		if c == child {
			return
		}
		ml.visitForNeedPaint(c, r)
	})
}

func (ml *MultiLayer) visitForNeedPaint(n Node, r *image.Rectangle) {
	ne := n.Embed()
	if ne.NeedsPaint() || ne.Hidden() || ne.NotPaintable() {
		return
	}
	if !n.Embed().Bounds.Overlaps(*r) {
		return
	}

	// if the childs union doesn't contain the rectangle, this node needs paint
	var u image.Rectangle
	n.Embed().IterChilds(func(c Node) {
		u = u.Union(c.Embed().Bounds)
	})
	if !r.In(u) {
		ne.MarkNeedsPaint() // highly recursive from here
		return
	}

	n.Embed().IterChilds(func(c Node) {
		ml.visitForNeedPaint(c, r)
	})
}
