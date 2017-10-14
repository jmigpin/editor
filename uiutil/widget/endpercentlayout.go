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
		if n.Prev().Prev() != nil {
			start = epl.endPercents[n.Prev().Prev()]
		}
		epl.endPercents[n.Prev()] = start + (end-start)/2
	}
	epl.endPercents[n] = end
}
func (epl *EndPercentLayout) InsertBefore(parent, n, mark Node) {
	epl.EmbedNode.InsertBefore(parent, n, mark)

	epl.lazyInit()

	// insert between siblings end percents
	start := 0.0
	end := 1.0
	if n.Prev() != nil {
		if n.Prev().Prev() != nil {
			start = epl.endPercents[n.Prev().Prev()]
		}
		end = epl.endPercents[n.Prev()]
		epl.endPercents[n.Prev()] = start + (end-start)/2
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
	epl.lazyInit()
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
	epl.lazyInit()
	epl.endPercents[child] = v
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

	epl.lazyInit()

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

func (epl *EndPercentLayout) ResizeEndPercent(node Node, ep, min float64) {
	// limit to siblings bounds
	start := 0.0
	if node.Prev() != nil {
		if node.Prev().Prev() != nil {
			start = epl.endPercents[node.Prev().Prev()]
		}
	}
	end := epl.endPercents[node]
	if ep < start+min {
		ep = start + min
	}
	if ep > end-min {
		ep = end - min
	}

	// if there is no other option, squash it
	if ep < start+min {
		ep = start + (end-start)/2
	}

	// resize
	if node.Prev() != nil {
		epl.SetChildEndPercent(node.Prev(), ep)
	}
}

func (epl *EndPercentLayout) AttemptToSwap(parent, node Node, ep, min float64) {
	n := node
	if n.Prev() != nil && n.Prev().Prev() != nil {
		start := epl.endPercents[n.Prev().Prev()]
		if ep < start-min {
			prev := node.Prev()
			epl.Remove(node)
			epl.InsertBefore(parent, node, prev)
			epl.ResizeEndPercent(n, ep, min)
		}
	}
	if n.Next() != nil {
		start := epl.endPercents[n]
		if ep > start+min {
			u := n.Next()
			epl.Remove(u)
			epl.InsertBefore(parent, u, n)
			if u.Prev() != nil {
				epl.SetChildEndPercent(u.Prev(), start)
			}
			epl.ResizeEndPercent(n, ep, min)
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
