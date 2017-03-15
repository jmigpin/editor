package uiutil

import "image"

// flex reference for ideas
// https://www.w3.org/TR/css-flexbox-1/

// Simple style box model. Not fool proof.
type Container struct {
	Bounds    image.Rectangle
	Childs    []*Container
	needPaint bool
	PaintFunc func()

	Style struct {
		Direction    Direction    // childs containers
		Distribution Distribution // child containers

		// Individual distribution
		Hidden          bool
		MainSize        *int
		CrossSize       *int
		DynamicMainSize func() int

		// end percent distribution
		EndPercent *float64 // between 0 and 1
	}
}

func (c *Container) AppendChilds(cs ...*Container) {
	c.Childs = append(c.Childs, cs...)
}
func (c *Container) InsertChild(cc *Container, index int) {
	u := make([]*Container, 0, len(c.Childs)+1)
	u = append(u, c.Childs[:index]...)
	u = append(u, cc)
	u = append(u, c.Childs[index:]...)
	c.Childs = u
}
func (c *Container) SwapChilds(c1, c2 *Container) {
	var a [2]int
	n := 0
	for i, ci := range []*Container{c1, c2} {
		for j, cj := range c.Childs {
			if cj == ci {
				a[i] = j
				n++
			}
		}
	}
	if n != 2 {
		panic("container not found in childs")
	}
	i, j := a[0], a[1]
	c.Childs[i], c.Childs[j] = c.Childs[j], c.Childs[i]
}
func (c *Container) RemoveChild(cc *Container) {
	for i, c2 := range c.Childs {
		if c2 == cc {
			u := make([]*Container, 0, len(c.Childs)-1)
			u = append(u, c.Childs[:i]...)
			u = append(u, c.Childs[i+1:]...)
			c.Childs = u
			return
		}
	}
}
func (c *Container) CalcChildsBounds() {
	if len(c.Childs) == 0 {
		return
	}
	switch c.Style.Distribution {
	case IndividualDistribution:
		c.individualDistribution()
	case EqualDistribution:
		c.equalDistribution()
	case EndPercentDistribution:
		c.endPercentDistribution()
	}
}
func (c *Container) individualDistribution() {
	dir := c.Style.Direction
	ms, me := c.mainStartEnd(dir)
	cs, ce := c.crossStartEnd(dir)
	mainSize := me - ms

	mainSizes := make(map[*Container]int, len(c.Childs))

	availableSize := func() int {
		used := 0
		for _, s := range mainSizes {
			used += s
		}
		rest := mainSize - used
		if rest < 0 {
			rest = 0
		}
		return rest
	}

	for _, child := range c.Childs {
		// hidden
		if child.Style.Hidden {
			mainSizes[child] = 0
			continue
		}
		// main fixed sizes
		if child.Style.MainSize != nil {
			mainSizes[child] = *child.Style.MainSize
			continue
		}
	}
	// dynamic main sizes
	for _, child := range c.Childs {
		_, ok := mainSizes[child]
		if ok {
			continue
		}
		if child.Style.DynamicMainSize != nil {
			mainSizes[child] = child.Style.DynamicMainSize()
		}
	}
	// distribute available space by those without a value set
	n := len(c.Childs) - len(mainSizes)
	if n > 0 {
		rest := availableSize()
		size := rest / n
		for _, child := range c.Childs {
			_, ok := mainSizes[child]
			if ok {
				continue
			}
			// avoid rounding errors not filling the last pixel
			n--
			if n == 0 {
				mainSizes[child] = rest
				break
			}

			rest -= size
			mainSizes[child] = size
		}
	}
	// set sizes
	cms := ms
	for _, child := range c.Childs {
		cme := cms + mainSizes[child]
		child.setMainStartEnd(dir, cms, cme)

		// cross fixed size
		ccs, cce := cs, ce
		if child.Style.CrossSize != nil {
			cce = ccs + *child.Style.CrossSize
		}
		child.setCrossStartEnd(dir, ccs, cce)

		// limit inside parent
		child.Bounds = child.Bounds.Intersect(c.Bounds)

		child.CalcChildsBounds()

		cms += mainSizes[child]
	}
}
func (c *Container) equalDistribution() {
	dir := c.Style.Direction
	ms, me := c.mainStartEnd(dir)
	cs, ce := c.crossStartEnd(dir)
	size := float64(me-ms) / float64(len(c.Childs)) // each child size
	for i, child := range c.Childs {
		cms := ms + int(float64(i)*size)
		cme := ms + int(float64(i+1)*size)
		if i == len(c.Childs)-1 {
			cme = me
		}
		child.setMainStartEnd(dir, cms, cme)
		child.setCrossStartEnd(dir, cs, ce)
		//fmt.Printf("container=%p, bounds=%v\n", child, child.Bounds)
		child.CalcChildsBounds()
	}
}
func (c *Container) endPercentDistribution() {
	dir := c.Style.Direction
	ms, me := c.mainStartEnd(dir)
	cs, ce := c.crossStartEnd(dir)
	mainSize := me - ms
	mainEnds := make(map[*Container]int, len(c.Childs))

	setEndPercentFromMainEnd := func(c2 *Container) {
		me := mainEnds[c2]
		ep := float64(me) / float64(mainSize)
		c2.Style.EndPercent = &ep
	}

	// end percents
	for _, child := range c.Childs {
		if child.Style.EndPercent != nil {
			p := float64(*child.Style.EndPercent)
			mainEnds[child] = int(float64(mainSize) * p)
		}
	}
	// end of childs not set
	for i, child := range c.Childs {
		_, ok := mainEnds[child]
		if !ok {
			start := 0
			if i > 0 {
				if i == len(c.Childs)-1 {
					// changed previous column to half
					u0 := 0
					if i >= 2 {
						u0 = mainEnds[c.Childs[i-2]]
					}
					u1 := mainEnds[c.Childs[i-1]]
					mainEnds[c.Childs[i-1]] = u0 + (u1-u0)/2
					setEndPercentFromMainEnd(c.Childs[i-1])
				}
				start = mainEnds[c.Childs[i-1]]
			}
			// size based on available range to next set child
			size := mainSize - start
			for j := i + 1; j < len(c.Childs); j++ {
				u, ok := mainEnds[c.Childs[j]]
				if ok {
					size = (u - start) / (j - i + 1)
					break
				}
			}
			mainEnds[child] = start + size
			setEndPercentFromMainEnd(child)
		}
	}
	// override last child to match bounds (expands to end)
	// avoids last pixel not being drawn due to rounding error
	if len(c.Childs) > 0 {
		lc := c.Childs[len(c.Childs)-1]
		mainEnds[lc] = mainSize
		setEndPercentFromMainEnd(lc)
	}
	// set sizes and end percents
	cms := ms
	for _, child := range c.Childs {
		cme := ms + mainEnds[child]
		child.setMainStartEnd(dir, cms, cme)
		child.setCrossStartEnd(dir, cs, ce)
		child.CalcChildsBounds()
		cms = cme
	}
}
func (c *Container) mainStartEnd(dir Direction) (int, int) {
	switch dir {
	case RowDirection:
		return c.Bounds.Min.X, c.Bounds.Max.X
	case ColumnDirection:
		return c.Bounds.Min.Y, c.Bounds.Max.Y
	}
	panic("!")
}
func (c *Container) setMainStartEnd(dir Direction, s, e int) {
	switch dir {
	case RowDirection:
		c.Bounds.Min.X = s
		c.Bounds.Max.X = e
		return
	case ColumnDirection:
		c.Bounds.Min.Y = s
		c.Bounds.Max.Y = e
		return
	}
	panic("!")
}
func (c *Container) crossStartEnd(dir Direction) (int, int) {
	switch dir {
	case RowDirection:
		return c.Bounds.Min.Y, c.Bounds.Max.Y
	case ColumnDirection:
		return c.Bounds.Min.X, c.Bounds.Max.X
	}
	panic("!")
}
func (c *Container) setCrossStartEnd(dir Direction, s, e int) {
	switch dir {
	case RowDirection:
		c.Bounds.Min.Y = s
		c.Bounds.Max.Y = e
		return
	case ColumnDirection:
		c.Bounds.Min.X = s
		c.Bounds.Max.X = e
		return
	}
	panic("!")
}
func (c *Container) PaintTree() {
	c.paint()
	for _, child := range c.Childs {
		child.PaintTree()
	}
}
func (c *Container) PaintTreeIfNeeded(fn func(*Container)) {
	if c.needPaint {
		c.PaintTree()
		fn(c) // call on top of the tree container being drawn
		return
	}
	for _, child := range c.Childs {
		child.PaintTreeIfNeeded(fn)
	}
}
func (c *Container) paint() {
	c.needPaint = false
	if c.PaintFunc != nil {
		c.PaintFunc()
	}
}
func (c *Container) NeedPaint() {
	c.needPaint = true
}

type Direction int

const (
	RowDirection Direction = iota
	ColumnDirection
)

type Distribution int

const (
	IndividualDistribution Distribution = iota
	EqualDistribution
	EndPercentDistribution
)
