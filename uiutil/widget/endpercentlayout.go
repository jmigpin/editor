package widget

import "image"

// End percent means that the child right X bound ends in the percent set.
// This ensures a change in the percentage of a middle child doesn't affect the bounds of the other childs (ex: like causing a small adjustment when resizing).

type EndPercentLayout struct {
	EmbedNode
	YAxis bool

	endPercents map[Node]float64 // between 0 and 1
}

func (epl *EndPercentLayout) lazyInit() {
	if epl.endPercents == nil {
		epl.endPercents = make(map[Node]float64)
	}
}

func (epl *EndPercentLayout) PushBack(parent, n Node) {
	epl.EmbedNode.PushBack(parent, n)

	epl.lazyInit()

	// share with last sibling
	start := 0.0
	end := 1.0
	if n.Prev() != nil {
		start = epl.ChildStartPercent(n.Prev())
		epl.endPercents[n.Prev()] = start + (end-start)/2
	}
	epl.endPercents[n] = end
}
func (epl *EndPercentLayout) InsertBefore(parent, n, mark Node) {
	epl.EmbedNode.InsertBefore(parent, n, mark)

	epl.lazyInit()

	// insert between siblings end percents
	start, end := 0.0, 1.0
	if n.Prev() != nil {
		start, end = epl.ChildPercents(n.Prev())
		epl.SetChildEndPercent(n.Prev(), start+(end-start)/2)
	}
	epl.endPercents[n] = end
}

func (epl *EndPercentLayout) Remove(n Node) {
	// have the previous node keep the space
	if n.Prev() != nil {
		u := epl.endPercents[n]
		epl.SetChildEndPercent(n.Prev(), u)
	}

	delete(epl.endPercents, n)
	epl.EmbedNode.Remove(n)
}

func (epl *EndPercentLayout) ChildEndPercent(child Node) float64 {
	v, ok := epl.endPercents[child]
	if !ok {
		panic("missing end percent")
	}
	return v
}
func (epl *EndPercentLayout) SetChildEndPercent(child Node, v float64) {
	if !epl.HasChild(child) {
		panic("not a child")
	}
	epl.endPercents[child] = v
}

func (epl *EndPercentLayout) ChildStartPercent(child Node) float64 {
	start := 0.0
	if child.Prev() != nil {
		v, ok := epl.endPercents[child.Prev()]
		if !ok {
			panic("missing end percent")
		}
		start = v
	}
	return start
}
func (epl *EndPercentLayout) SetChildStartPercent(child Node, v float64) {
	if !epl.HasChild(child) {
		panic("not a child")
	}
	if child.Prev() != nil {
		epl.endPercents[child.Prev()] = v
	}
}

func (epl *EndPercentLayout) ChildPercents(child Node) (float64, float64) {
	start := epl.ChildStartPercent(child)
	end := epl.ChildEndPercent(child)
	return start, end
}

func (epl *EndPercentLayout) Measure(hint image.Point) image.Point {
	// TODO: measure childs - this measure is primarily working for an expanded node

	return image.Point{10, 10}
}

func (epl *EndPercentLayout) CalcChildsBounds() {
	childs := epl.Childs()
	if len(childs) == 0 {
		return
	}

	// translate axis
	xya := XYAxis{epl.YAxis}
	bounds := epl.Bounds()
	abounds := xya.Rectangle(&bounds)

	// set sizes
	cxs := abounds.Min.X
	for _, child := range childs {
		ep, _ := epl.endPercents[child]

		xEnd := int(ep * float64(abounds.Dx()))

		var r image.Rectangle
		cxe := abounds.Min.X + xEnd
		r.Min = image.Point{cxs, abounds.Min.Y}
		r.Max = image.Point{cxe, abounds.Max.Y}
		cxs = cxe

		// fix last child for rounding errors
		if child == epl.LastChild() {
			r.Max.X = abounds.Max.X
		}

		// translate axis
		r2 := xya.Rectangle(&r)

		r3 := r2.Intersect(epl.Bounds())
		child.SetBounds(&r3)
		child.CalcChildsBounds()
	}
}

func (epl *EndPercentLayout) ResizeEndPercent(node Node, percent float64, percentIsMin bool, pad float64) {
	epl.resizeChild(node, percent, percentIsMin, pad)
}
func (epl *EndPercentLayout) ResizeEndPercentWithPush(node Node, percent float64, percentIsMin bool, pad float64) {
	epl.resizeChildWithPush(node, percent, percentIsMin, pad)
}
func (epl *EndPercentLayout) ResizeEndPercentWithSwap(parent, node Node, percent float64, percentIsMin bool, pad float64) {
	epl.attemptToSwap(parent, node, percent, percentIsMin, pad)
	epl.resizeChild(node, percent, percentIsMin, pad)
}

func (epl *EndPercentLayout) resizeChild(node Node, percent float64, percentIsMin bool, pad float64) {
	min := 0.0
	max := 1.0
	if percentIsMin {
		if node.Prev() != nil {
			min = epl.ChildStartPercent(node.Prev())
		}
		max = epl.ChildEndPercent(node)
	} else {
		min = epl.ChildStartPercent(node)
		if node.Next() != nil {
			max = epl.ChildEndPercent(node.Next())
		}
	}

	// limit with some pad
	if percent < min+pad {
		percent = min + pad
	}
	if percent > max-pad {
		percent = max - pad
	}

	// squash it in the middle
	if percent < min+pad {
		percent = min + (max-min)/2
	}

	// resize
	if percentIsMin {
		if node != epl.FirstChild() {
			epl.SetChildStartPercent(node, percent)
		}
	} else {
		if node != epl.LastChild() {
			epl.SetChildEndPercent(node, percent)
		}
	}
}

func (epl *EndPercentLayout) resizeChildWithPush(node Node, percent float64, percentIsMin bool, pad float64) {
	if percentIsMin {
		// resize siblings up
		if node.Prev() != nil {
			min := epl.ChildStartPercent(node.Prev())
			if percent < min+pad {
				diff := (min + pad) - percent
				epl.resizeChildWithPush(node.Prev(), min-diff, percentIsMin, pad)
				min = epl.ChildStartPercent(node.Prev())
				if percent < min+pad {
					percent = min + pad
				}
			}
		}
		// resize siblings down
		if node.Next() != nil {
			max := epl.ChildEndPercent(node)
			if percent > max-pad {
				diff := percent - (max - pad)
				epl.resizeChildWithPush(node.Next(), max+diff, percentIsMin, pad)
				max = epl.ChildEndPercent(node)
				if percent > max-pad {
					percent = max - pad
				}
			}
		}
	} else {
		panic("TODO: not used yet so not implemented")
	}

	epl.resizeChild(node, percent, percentIsMin, pad)
}

func (epl *EndPercentLayout) attemptToSwap(parent, node Node, percent float64, percentIsMin bool, pad float64) {
	if percentIsMin {
		if node.Prev() != nil && node.Prev().Prev() != nil {
			min := epl.ChildStartPercent(node.Prev())
			if percent < min {
				prev := node.Prev()
				epl.Remove(node)
				epl.InsertBefore(parent, node, prev)
			}
		}
		if node.Next() != nil {
			max := epl.ChildEndPercent(node)
			if percent > max {
				nextnext := node.Next().Next()
				epl.Remove(node)
				if nextnext == nil {
					epl.PushBack(parent, node)
				} else {
					epl.InsertBefore(parent, node, nextnext)
				}
			}
		}
	} else {
		if node.Prev() != nil {
			min := epl.ChildStartPercent(node)
			if percent < min {
				start := epl.ChildStartPercent(node.Prev())
				end := epl.ChildStartPercent(node)

				prev := node.Prev()
				epl.Remove(node)
				epl.InsertBefore(parent, node, prev)

				epl.SetChildStartPercent(node, start)
				epl.SetChildEndPercent(prev, end)
			}
		}
		if node.Next() != nil && node.Next().Next() != nil {
			max := epl.ChildEndPercent(node.Next())
			if percent > max {
				next := node.Next()
				start := epl.ChildStartPercent(node)
				end := epl.ChildEndPercent(next)

				nextnext := node.Next().Next()
				epl.Remove(node)
				if nextnext == nil {
					epl.PushBack(parent, node)
				} else {
					epl.InsertBefore(parent, node, nextnext)
				}

				epl.SetChildStartPercent(next, start)
				epl.SetChildEndPercent(next, end)
			}
		}
	}
}

func (epl *EndPercentLayout) MaximizeEndPercentNode(node Node, min float64) {
	childs := epl.Childs()
	n := len(childs)
	ep := 0.0
	for i, c := range childs {
		if c == node {
			ep = 1.0 - min*float64(n-(i+1))
		} else {
			ep += min
		}
		epl.SetChildEndPercent(c, ep)
	}
}
