package widget

import (
	"image"
)

// First child is bottom layer.
type MultiLayer struct {
	ENode

	BgLayer        *BgLayer
	SeparatorLayer *ENode
	FloatLayer1    *FloatLayer
}

func NewMultiLayer() *MultiLayer {
	ml := &MultiLayer{}

	ml.BgLayer = &BgLayer{ml: ml}
	ml.SeparatorLayer = &ENode{}
	ml.FloatLayer1 = &FloatLayer{ml: ml}

	// order matters
	layers := []Node{
		ml.BgLayer,
		ml.SeparatorLayer,
		ml.FloatLayer1,
	}
	for _, u := range layers {
		if u == nil {
			continue
		}
		ml.ENode.InsertBefore(u, nil)

		// allow drag events to fall through to lower layers nodes
		u.Embed().Marks.Add(MarkNotDraggable)
	}

	return ml
}

func (ml *MultiLayer) InsertBefore(col Node, next *EmbedNode) {
	panic("nodes should be inserted into one of the layers directly")
}

func (ml *MultiLayer) PaintMarked() image.Rectangle {
	vnodes := ml.FloatLayer1.visibleNodes()
	if len(vnodes) > 0 {
		// mark floatlayer visible nodes as needingpaint when bg nodes are painted
		for _, n := range vnodes {
			if intersectingNodeNeedingPaintExists(ml.BgLayer, n.Embed().Bounds) {
				n.Embed().MarkNeedsPaint()

				// mark bglayer nodes as needing paint if intersecting with floatlayer visible nodes
				ml.BgLayer.RectNeedsPaint(n.Embed().Bounds)
			}
		}

	}

	return ml.ENode.PaintMarked()
}

//----------

type BgLayer struct {
	ENode
	ml *MultiLayer
}

func (bgl *BgLayer) RectNeedsPaint(r image.Rectangle) {
	markIntersectingNodesNotNeedingPaint(bgl, r)
}

//----------

type FloatLayer struct {
	ENode
	ml *MultiLayer
}

func (fl *FloatLayer) OnChildMarked(child Node, newMarks Marks) {
	if newMarks.HasAny(MarkNeedsLayout | MarkChildNeedsLayout) {
		fl.MarkNeedsLayout()
	}
}

func (fl *FloatLayer) Layout() {
	nodes := fl.visibleNodes()
	for _, n := range nodes {
		n.Embed().MarkNeedsPaint()
		fl.ml.BgLayer.RectNeedsPaint(n.Embed().Bounds)
	}
}

func (fl *FloatLayer) visibleNodes() []Node {
	return visibleChildNodes(fl)
}

//----------

func visibleChildNodes(node Node) []Node {
	z := []Node{}
	node.Embed().IterateWrappers2(func(child Node) {
		if !child.Embed().Marks.HasAny(MarkForceZeroBounds) {
			z = append(z, child)
		}
	})
	return z
}

//----------

func intersectingNodeNeedingPaintExists(node Node, r image.Rectangle) bool {
	found := false
	node.Embed().IterateWrappers(func(child Node) bool {
		ce := child.Embed()
		if ce.Bounds.Overlaps(r) {
			if ce.Marks.HasAny(MarkNeedsPaint) {
				found = true
			} else if ce.Marks.HasAny(MarkChildNeedsPaint) {
				found = intersectingNodeNeedingPaintExists(child, r)
			}
		}
		return !found // continue while not found
	})
	return found
}

//----------

func markIntersectingNodesNotNeedingPaint(node Node, r image.Rectangle) image.Rectangle {
	u := image.Rectangle{}
	node.Embed().IterateWrappers2(func(child Node) {
		ce := child.Embed()
		if ce.Bounds.Overlaps(r) {
			if !ce.Marks.HasAny(MarkNeedsPaint) {

				// improve selection with subchilds
				if r.In(ce.Bounds) {
					w := markIntersectingNodesNotNeedingPaint(child, r)
					u = u.Union(w)
					if r.In(w) {
						return
					}
				}

				u = u.Union(ce.Bounds)
				ce.MarkNeedsPaint()
			}
		}
	})
	return u
}
