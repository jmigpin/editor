package widget

import (
	"container/list"
	"image"
)

// First/last child is the bottom/top layer.
type MultiLayer struct {
	ENode

	BgLayer        *BgLayer
	SeparatorLayer *ENode
	ContextLayer   *FloatLayer
	MenuLayer      *FloatLayer

	rects list.List
}

func NewMultiLayer() *MultiLayer {
	ml := &MultiLayer{}

	ml.BgLayer = &BgLayer{ml: ml}
	ml.SeparatorLayer = &ENode{}
	ml.ContextLayer = &FloatLayer{ml: ml}
	ml.MenuLayer = &FloatLayer{ml: ml}

	// order matters
	ml.Append(
		ml.BgLayer,
		ml.SeparatorLayer,
		ml.ContextLayer,
		ml.MenuLayer,
	)

	ml.Iterate2(func(en *EmbedNode) {
		// allow drag events to fall through to lower layers nodes
		en.AddMarks(MarkNotDraggable)

	})

	return ml
}

func (ml *MultiLayer) InsertBefore(col Node, next *EmbedNode) {
	panic("nodes should be inserted into one of the layers directly")
}

//----------

func (ml *MultiLayer) PaintMarked() image.Rectangle {
	ml.markAddedRects()
	ml.markFloatLayers()
	return ml.ENode.PaintMarked()
}

//----------

func (ml *MultiLayer) AddMarkRect(r image.Rectangle) {
	ml.rects.PushBack(&r)
}

func (ml *MultiLayer) markAddedRects() {
	for elem := ml.rects.Front(); elem != nil; elem = elem.Next() {
		r := elem.Value.(*image.Rectangle)
		ml.markRect(nil, *r)
	}
	ml.rects = list.List{}
}

//----------

func (ml *MultiLayer) markFloatLayers() {
	ml.IterateWrappers2(func(n Node) {
		if fl, ok := n.(*FloatLayer); ok {
			ml.markVisibleNodes(fl)
		}
	})
}

func (ml *MultiLayer) markVisibleNodes(fl *FloatLayer) {
	vnodes := fl.visibleNodes()
	for _, n := range vnodes {
		ne := n.Embed()
		if ml.rectNeedsPaint(ne.Bounds) {
			ne.MarkNeedsPaint()
			ml.markRect(fl, ne.Bounds)
		}
	}
}

//----------

func (ml *MultiLayer) rectNeedsPaint(r image.Rectangle) bool {
	found := false
	ml.IterateWrappers(func(layer Node) bool {
		found = intersectingNodeNeedingPaintExists(layer, r)
		return !found // continue if not found
	})
	return found
}

func (ml *MultiLayer) markRect(callLayer Node, r image.Rectangle) {
	ml.IterateWrappers2(func(layer Node) {
		if layer != callLayer { // performance
			markIntersectingNodesNotNeedingPaint(layer, r)
		}
	})
}

//----------

type BgLayer struct {
	ENode
	ml *MultiLayer
}

//----------

type FloatLayer struct {
	ENode
	ml *MultiLayer
}

func (fl *FloatLayer) OnChildMarked(child Node, newMarks Marks) {
	// force float layer to recalc childs bounds to avoid childs having to consult the floatlayer bounds
	if newMarks.HasAny(MarkNeedsLayout | MarkChildNeedsLayout) {
		fl.MarkNeedsLayout()
	}
	//if newMarks.HasAny(MarkNeedsPaint | MarkChildNeedsPaint) {
	//	log.Printf("float needs paint: %p", fl)
	//	child.Embed().MarkNeedsPaint()
	//}
}

func (fl *FloatLayer) visibleNodes() []Node {
	return visibleChildNodes(fl)
}

//----------

func visibleChildNodes(node Node) []Node {
	z := []Node{}
	node.Embed().IterateWrappers2(func(child Node) {
		if !child.Embed().HasAnyMarks(MarkForceZeroBounds) {
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
			if ce.HasAnyMarks(MarkNeedsPaint) {
				found = true
			} else if ce.HasAnyMarks(MarkChildNeedsPaint) {
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
			if !ce.HasAnyMarks(MarkNeedsPaint) {

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
