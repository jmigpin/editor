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

	// share with previous sibling, or with the next one if becoming first child
	var end float64
	if n.Prev() != nil {
		start := epl.ChildStartPercent(n.Prev())
		end = epl.ChildEndPercent(n.Prev())
		epl.SetChildEndPercent(n.Prev(), start+(end-start)/2)
	} else {
		// first child
		w := epl.ChildEndPercent(mark)
		end = w / 2
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

func (epl *EndPercentLayout) ResizeEndPercent(node Node, percent float64, percentIsMin bool, minPerc float64) {
	epl.resizeChild(node, percent, percentIsMin, minPerc)
}
func (epl *EndPercentLayout) ResizeEndPercentWithPush(node Node, percent float64, percentIsMin bool, minPerc float64) {
	epl.resizeChildWithPush(node, percent, percentIsMin, minPerc)
}
func (epl *EndPercentLayout) ResizeEndPercentWithSwap(parent, node Node, percent float64, percentIsMin bool, minPerc float64) {
	epl.resizeChildWithAttemptToSwap(parent, node, percent, percentIsMin, minPerc)
}

func (epl *EndPercentLayout) resizeChild(node Node, percent float64, percentIsMin bool, minPerc float64) {
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

	// check limits
	if percent < min+minPerc {
		percent = min + minPerc
	}
	if percent > max-minPerc {
		percent = max - minPerc
	}

	// squash it in the middle
	if percent < min+minPerc {
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

func (epl *EndPercentLayout) resizeChildWithPush(node Node, percent float64, percentIsMin bool, minPerc float64) {
	if percentIsMin {
		if node.Prev() != nil {
			min := epl.ChildStartPercent(node.Prev())
			if percent < min+minPerc {
				diff := (min + minPerc) - percent
				epl.resizeChildWithPush(node.Prev(), min-diff, percentIsMin, minPerc)
				min = epl.ChildStartPercent(node.Prev())
				if percent < min+minPerc {
					percent = min + minPerc
				}
			}
		}
		if node.Next() != nil {
			max := epl.ChildEndPercent(node)
			if percent > max-minPerc {
				diff := percent - (max - minPerc)
				epl.resizeChildWithPush(node.Next(), max+diff, percentIsMin, minPerc)
				max = epl.ChildEndPercent(node)
				if percent > max-minPerc {
					percent = max - minPerc
				}
			}
		}
	} else {
		if node.Prev() != nil {
			min := epl.ChildStartPercent(node)
			if percent < min+minPerc {
				diff := (min + minPerc) - percent
				epl.resizeChildWithPush(node.Prev(), min-diff, percentIsMin, minPerc)
				min = epl.ChildStartPercent(node)
				if percent < min+minPerc {
					percent = min + minPerc
				}
			}
		}
		if node.Next() != nil {
			max := epl.ChildEndPercent(node.Next())
			if percent > max-minPerc {
				diff := percent - (max - minPerc)
				epl.resizeChildWithPush(node.Next(), max+diff, percentIsMin, minPerc)
				max = epl.ChildEndPercent(node.Next())
				if percent > max-minPerc {
					percent = max - minPerc
				}
			}
		}
	}

	epl.resizeChild(node, percent, percentIsMin, minPerc)
}

func (epl *EndPercentLayout) resizeChildWithAttemptToSwap(parent, node Node, percent float64, percentIsMin bool, minPerc float64) {
	// n0,n1,n2,n3,n4: moving n2

	n1 := node.Prev()
	n2 := node
	n3 := n2.Next()
	var n0 Node
	if n1 != nil && n1.Prev() != nil {
		n0 = n1.Prev()
	}
	var n4 Node
	if n3 != nil && n3.Next() != nil {
		n4 = n3.Next()
	}

	ep := epl.ChildEndPercent
	setEp := epl.SetChildEndPercent

	if percentIsMin {
		if n1 != nil && n0 != nil {
			min := ep(n0)
			if percent < min {
				n2.Swap(n1)
				setEp(n1, ep(n2))
				setEp(n2, ep(n0))
				// n0 ep will be resized
			}
		}
		if n3 != nil {
			max := ep(n2)
			if percent > max {
				n2.Swap(n3)
				if n1 == nil {
					setEp(n2, ep(n3))
					// n3 ep will be resized
				} else {
					setEp(n1, ep(n2))
					setEp(n2, ep(n3))
					// n3 ep will be resized
				}
			}
		}
	} else {
		if n1 != nil {
			min := ep(n1)
			if percent < min {
				n2.Swap(n1)
				if n3 == nil {
					setEp(n1, ep(n2))
					// n2 ep will be resized
				} else {
					// n2 ep will be resized
				}
			}
		}
		if n3 != nil && n4 != nil {
			max := ep(n3)
			if percent > max {
				n2.Swap(n3)
				setEp(n2, ep(n4))
				// n4 ep will be resized
			}
		}
	}

	epl.resizeChild(node, percent, percentIsMin, minPerc)
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
