package widget

import (
	"image"
)

type BoxLayout struct {
	EmbedNode
	YAxis bool

	flex map[Node]XYAxisBoolPair // used in measuring+calcchilds, has priority over fill
	fill map[Node]XYAxisBoolPair // used only in calcchilds
}

func NewBoxLayout() *BoxLayout {
	bl := &BoxLayout{
		flex: make(map[Node]XYAxisBoolPair),
		fill: make(map[Node]XYAxisBoolPair),
	}
	return bl
}

func (bl *BoxLayout) Measure(hint image.Point) image.Point {
	bounds := bl.childsBounds(hint, true)
	xya := &XYAxis{bl.YAxis}
	var max image.Point
	for _, b := range bounds {
		s := b.Size()
		size := xya.Point(&s)
		max.X += size.X
		if size.Y > max.Y {
			max.Y = size.Y
		}
	}
	return xya.Point(&max)
}

func (bl *BoxLayout) CalcChildsBounds() {
	bounds := bl.childsBounds(bl.Bounds.Size(), false)

	// set bounds
	bl.IterChilds(func(child Node) {
		b := bounds[child]
		r := b.Add(bl.Bounds.Min)

		r3 := r.Intersect(bl.Bounds)
		child.Embed().Bounds = r3
		child.CalcChildsBounds()
	})
}

func (bl *BoxLayout) childsBounds(max image.Point, measure bool) map[Node]image.Rectangle {
	xya := &XYAxis{bl.YAxis}
	max2 := xya.Point(&max)
	sizes := make(map[Node]image.Point, bl.ChildsLen())

	// count flex/fill
	nFlexX := 0
	nFillX := 0
	var lastFlexXNode, lastFillXNode Node
	bl.IterChilds(func(child Node) {
		bp := xya.BoolPair(bl.flex[child])
		if bp.X {
			nFlexX++
			lastFlexXNode = child
		}
		if !measure {
			bp := xya.BoolPair(bl.fill[child])
			if bp.X {
				nFillX++
				lastFillXNode = child
			}
		}
	})

	// x fills are only considered if there are no x flexes (priority for flex)
	flexingX := nFlexX > 0
	nX := nFlexX
	fillingX := false
	if !flexingX && !measure && nFillX > 0 {
		fillingX = true
		nX = nFillX
	}

	// measure non-flexible childs first to get remaining space
	available := max2
	bl.IterChilds(func(child Node) {
		bp := xya.BoolPair(bl.flex[child])
		bp2 := xya.BoolPair(bl.fill[child])
		if (!flexingX && !fillingX) || (flexingX && !bp.X) || (fillingX && !bp2.X) {
			// flex: -X-Y
			m0 := child.Measure(xya.Point(&available))
			m := xya.Point(&m0)

			// flex: -X+Y
			if bp.Y || (!measure && bp2.Y) {
				m.Y = available.Y
			}

			sizes[child] = m
			available.X -= m.X
			if available.X < 0 {
				available.X = 0
			}
		}
	})

	// x flex childs
	{
		// divide remaining space among the flexible childs
		share := available
		if nX > 0 {
			share.X = available.X / nX
		}

		// measure flexible childs
		bl.IterChilds(func(child Node) {
			bp := xya.BoolPair(bl.flex[child])
			bp2 := xya.BoolPair(bl.fill[child])
			if (flexingX && bp.X) || (fillingX && bp2.X) {
				var m image.Point

				// flex: +X+Y
				if bp.Y || (!measure && bp2.Y) {
					m = share
				} else {
					// flex: +X-Y
					m0 := child.Measure(xya.Point(&share))
					m = xya.Point(&m0)
					m.X = share.X
				}

				// correct rounding errors on last node
				if child == lastFlexXNode {
					m.X = available.X - (share.X * (nX - 1))
				}

				sizes[child] = m
			}
		})
	}

	// setup bounds
	bounds := make(map[Node]image.Rectangle, bl.ChildsLen())
	x := 0
	bl.IterChilds(func(child Node) {
		size := sizes[child]
		r := image.Rect(x, 0, x+size.X, size.Y)
		bounds[child] = xya.Rectangle(&r)
		x += size.X
	})
	return bounds
}

func (bl *BoxLayout) SetChildFlex(node Node, x, y bool) {
	bl.flex[node] = XYAxisBoolPair{x, y}
}

func (bl *BoxLayout) SetChildFill(node Node, x, y bool) {
	bl.fill[node] = XYAxisBoolPair{x, y}
}
