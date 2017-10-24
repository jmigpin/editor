package widget

import "image"

// TODO: improve tree search performance with a cache (calcchilds seed?)

type FlowLayout struct {
	EmbedNode
	YAxis bool
}

func NewFlowLayout() *FlowLayout {
	fl := &FlowLayout{}
	fl.SetWrapper(fl)
	return fl
}
func (fl *FlowLayout) Measure(hint image.Point) image.Point {
	sizes := fl.measureChildsSizes(hint)
	xya := &XYAxis{fl.YAxis}
	var max image.Point
	for _, s := range sizes {
		s2 := xya.Point(s)
		max.X += s2.X
		if s2.Y > max.Y {
			max.Y = s2.Y
		}
	}
	return *xya.Point(&max)
}

func (fl *FlowLayout) measureChildsSizes(max image.Point) map[Node]*image.Point {
	childs := fl.Childs()
	xya := &XYAxis{fl.YAxis}
	max2 := xya.Point(&max)

	sizes := make(map[Node]*image.Point, len(childs))

	// measure childs not expanding in X
	max3 := *max2
	nExpandX := 0
	for _, child := range childs {
		if hasAxisExpandXInTree(child, xya) {
			// expand: +X
			nExpandX++
		} else {
			// expand: -X-Y
			m0 := child.Measure(*xya.Point(&max3))
			m := xya.Point(&m0)

			// expand: -X+Y
			if hasAxisExpandYInTree(child, xya) {
				m.Y = max2.Y
			}

			sizes[child] = m
			max3.X -= m.X
			if max3.X < 0 {
				max3.X = 0
			}
		}
	}

	// space share for childs expanding in X
	xShare := 0
	availableX := max3.X
	if nExpandX > 0 {
		xShare = availableX / nExpandX
	}

	for _, child := range childs {
		_, ok := sizes[child]
		if ok {
			continue
		}
		if hasAxisExpandYInTree(child, xya) {
			// expand: +X+Y
			sizes[child] = &image.Point{xShare, max2.Y}
		} else {
			// expand: +X-Y
			max4 := image.Point{xShare, max2.Y}
			m0 := child.Measure(*xya.Point(&max4))
			m := xya.Point(&m0)
			m.X = xShare
			sizes[child] = m
		}
	}

	// translate axis
	for c, s := range sizes {
		sizes[c] = xya.Point(s)
	}

	return sizes
}

func axisExpandX(n Node, xya *XYAxis) bool {
	x0, y0 := n.Expand()
	x, _ := xya.BoolPair(x0, y0)
	return x
}
func axisExpandY(n Node, xya *XYAxis) bool {
	x0, y0 := n.Expand()
	_, y := xya.BoolPair(x0, y0)
	return y
}
func axisFillX(n Node, xya *XYAxis) bool {
	x0, y0 := n.Fill()
	x, _ := xya.BoolPair(x0, y0)
	return x
}
func axisFillY(n Node, xya *XYAxis) bool {
	x0, y0 := n.Fill()
	_, y := xya.BoolPair(x0, y0)
	return y
}

func hasAxisExpandXInTree(n Node, xya *XYAxis) bool {
	if axisExpandX(n, xya) {
		return true
	}
	for _, c := range n.Childs() {
		v := hasAxisExpandXInTree(c, xya)
		if v {
			return true
		}
	}
	return false
}
func hasAxisExpandYInTree(n Node, xya *XYAxis) bool {
	if axisExpandY(n, xya) {
		return true
	}
	for _, c := range n.Childs() {
		v := hasAxisExpandYInTree(c, xya)
		if v {
			return true
		}
	}
	return false
}

func hasAxisFillXInTree(n Node, xya *XYAxis) bool {
	if axisFillX(n, xya) {
		return true
	}
	for _, c := range n.Childs() {
		v := hasAxisFillXInTree(c, xya)
		if v {
			return true
		}
	}
	return false
}
func hasAxisFillYInTree(n Node, xya *XYAxis) bool {
	if axisFillY(n, xya) {
		return true
	}
	for _, c := range n.Childs() {
		v := hasAxisFillYInTree(c, xya)
		if v {
			return true
		}
	}
	return false
}

func (fl *FlowLayout) CalcChildsBounds() {
	childs := fl.Childs()
	bounds := fl.Bounds()
	max := image.Point{bounds.Dx(), bounds.Dy()}
	sizes0 := fl.measureChildsSizes(max)

	// translate axis
	xya := &XYAxis{fl.YAxis}
	abounds := xya.Rectangle(&bounds)
	sizes := make(map[Node]*image.Point, len(sizes0))
	for c, s := range sizes0 {
		sizes[c] = xya.Point(s)
	}

	// how many are filling in X
	nFillX := 0
	availableX := 0
	for _, child := range childs {
		if hasAxisFillXInTree(child, xya) {
			nFillX++
			size := sizes[child]
			availableX += size.X
		}
	}

	if nFillX > 0 {
		// get used X
		usedX := 0
		for _, child := range childs {
			size := sizes[child]
			usedX += size.X
		}
		availableX := abounds.Dx() - usedX

		// distribute among those that have the fill flag
		if availableX > 0 {
			xShare := availableX / nFillX
			for _, child := range childs {
				if hasAxisFillXInTree(child, xya) {
					size := sizes[child]
					size.X += xShare
					sizes[child] = size
				}
			}
		}
	}

	// set bounds
	cxs := abounds.Min.X
	for _, child := range childs {
		size := sizes[child]

		// Use measured Y, aligned to the top
		if hasAxisFillYInTree(child, xya) {
			size.Y = abounds.Dy()
		}

		var r image.Rectangle
		cxe := cxs + size.X
		r.Min = image.Point{cxs, abounds.Min.Y}
		r.Max = image.Point{cxe, abounds.Min.Y + size.Y}
		cxs = cxe

		// fix last child for rounding errors if some child expanded in X
		if child == fl.LastChild() && hasAxisFillXInTree(child, xya) {
			r.Max.X = abounds.Max.X
		}

		// translate axis
		r2 := xya.Rectangle(&r)

		r3 := r2.Intersect(fl.Bounds())

		child.SetBounds(&r3)
		child.CalcChildsBounds()
	}
}
