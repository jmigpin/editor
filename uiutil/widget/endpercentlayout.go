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
func (epl *EndPercentLayout) Remove(n Node) {
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

	// define end percents missing
	for _, child := range childs {
		ep, ok := epl.endPercents[child]
		if ok {
			continue
		}

		// previous end percent
		prevEP := 0.0
		if child.Prev() != nil {
			prevEP, _ = epl.endPercents[child.Prev()]
		}

		// next end percent
		nextEP := 1.0
		n := 1
		for c := child.Next(); c != nil; c = c.Next() {
			n++
			ep, ok = epl.endPercents[c]
			if ok {
				nextEP = ep
				break
			}
		}

		// share for the n childs not defined
		share := (nextEP - prevEP) / float64(n)

		// appending to a child at endpercent 1.0 - share space
		if share == 0 && child == epl.LastChild() && child.Prev() != nil {
			start := 0.0
			if child.Prev().Prev() != nil {
				start, _ = epl.endPercents[child.Prev().Prev()]
			}
			end, _ := epl.endPercents[child.Prev()]
			share = (end - start) / 2
			prevEP = start + share
			epl.endPercents[child.Prev()] = prevEP
		}

		// set share
		epl.endPercents[child] = prevEP + share
	}

	// set sizes and end percents
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
