package uiutil

// flex reference for ideas
// https://www.w3.org/TR/css-flexbox-1/

// Simple style box model. Not fool proof.

func SimpleBoxModelCalcChildsBounds(c *Container) {
	sm := SimpleBoxModel{}
	sm.CalcChildsBounds(c)
}

type SimpleBoxModel struct {
}

func (bm *SimpleBoxModel) CalcChildsBounds(c *Container) {
	if c.NChilds() == 0 {
		return
	}
	switch c.Style.Distribution {
	case IndividualDistribution:
		bm.individualDistribution(c)
	case EqualDistribution:
		bm.equalDistribution(c)
	case EndPercentDistribution:
		bm.endPercentDistribution(c)
	}
	// run childs calc callbacks
	for _, cc := range c.Childs() {
		if cc.OnCalcFunc != nil {
			cc.OnCalcFunc()
		}
	}
}

func (bm *SimpleBoxModel) individualDistribution(c *Container) {
	se := initStartEnd(c, &c.Style)
	ms, me := se.mainStartEnd()
	cs, ce := se.crossStartEnd()
	mainSize := me - ms
	mainSizes := make(map[*Container]int, c.NChilds())

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

	for _, child := range c.Childs() {
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
	for _, child := range c.Childs() {
		_, ok := mainSizes[child]
		if ok {
			continue
		}
		if child.Style.DynamicMainSize != nil {
			mainSizes[child] = child.Style.DynamicMainSize()
		}
	}
	// distribute available space by those without a value set
	n := c.NChilds() - len(mainSizes)
	if n > 0 {
		rest := availableSize()
		size := rest / n
		for _, child := range c.Childs() {
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
	for _, child := range c.Childs() {
		cme := cms + mainSizes[child]

		childSE := initStartEnd(child, &c.Style) // parent style
		childSE.setMainStartEnd(cms, cme)

		// cross fixed size
		ccs, cce := cs, ce
		if child.Style.CrossSize != nil {
			cce = ccs + *child.Style.CrossSize
		}
		childSE.setCrossStartEnd(ccs, cce)

		// limit to parent bounds
		child.Bounds = child.Bounds.Intersect(c.Bounds)

		child.CalcChildsBounds()
		cms += mainSizes[child]
	}
}

func (bm *SimpleBoxModel) equalDistribution(c *Container) {
	se := initStartEnd(c, &c.Style)
	ms, me := se.mainStartEnd()
	cs, ce := se.crossStartEnd()
	size := float64(me-ms) / float64(c.NChilds()) // each child size
	i := 0
	for _, child := range c.Childs() {
		cms := ms + int(float64(i)*size)
		cme := ms + int(float64(i+1)*size)
		if i == c.NChilds()-1 {
			cme = me
		}
		childSE := initStartEnd(child, &c.Style) // parent style
		childSE.setMainStartEnd(cms, cme)
		childSE.setCrossStartEnd(cs, ce)
		child.CalcChildsBounds()
		i++
	}
}

func (bm *SimpleBoxModel) endPercentDistribution(c *Container) {
	se := initStartEnd(c, &c.Style)
	ms, me := se.mainStartEnd()
	cs, ce := se.crossStartEnd()
	mainSize := me - ms
	mainEnds := make(map[*Container]int, c.NChilds())

	min := 20 // mim for calculations, all values trimmed to parent bounds at end
	if mainSize < min {
		mainSize = min
	}

	setEndPercentFromMainEnd := func(c2 *Container) {
		me := mainEnds[c2]
		ep := float64(me) / float64(mainSize)
		c2.Style.EndPercent = &ep
	}

	// end percents
	for _, child := range c.Childs() {
		if child.Style.EndPercent != nil {
			p := float64(*child.Style.EndPercent)
			mainEnds[child] = int(float64(mainSize) * p)
		}
	}
	// end of childs not set
	i := 0
	for _, child := range c.Childs() {
		_, ok := mainEnds[child]
		if !ok {
			start := 0
			if i > 0 {
				if i == c.NChilds()-1 {
					// changed previous column to half
					u0 := 0
					if i >= 2 {
						u0 = mainEnds[child.PrevSibling().PrevSibling()]
					}
					u1 := mainEnds[child.PrevSibling()]
					mainEnds[child.PrevSibling()] = u0 + (u1-u0)/2
					setEndPercentFromMainEnd(child.PrevSibling())
				}
				start = mainEnds[child.PrevSibling()]
			}
			// size based on available range to next set child
			size := mainSize - start
			j := 0
			for child2 := child.NextSibling(); child2 != nil; child2 = child2.NextSibling() {
				u, ok := mainEnds[child2]
				if ok {
					size = (u - start) / (j - i + 1)
					break
				}
				j++
			}
			mainEnds[child] = start + size
			setEndPercentFromMainEnd(child)
		}
		i++
	}
	// override last child to match bounds (expands to end)
	// avoids last pixel not being drawn due to rounding error
	if c.NChilds() > 0 {
		lc := c.LastChild()
		mainEnds[lc] = mainSize
		setEndPercentFromMainEnd(lc)
	}
	// set sizes and end percents
	cms := ms
	for _, child := range c.Childs() {
		cme := ms + mainEnds[child]

		childSE := initStartEnd(child, &c.Style) // parent style
		childSE.setMainStartEnd(cms, cme)
		childSE.setCrossStartEnd(cs, ce)

		// limit to parent bounds
		child.Bounds = child.Bounds.Intersect(c.Bounds)

		child.CalcChildsBounds()
		cms = cme
	}
}

// Allows setup of start/end depending on a row/column direction
type StartEnd struct {
	c     *Container
	style *Style
}

func initStartEnd(c *Container, style *Style) StartEnd {
	return StartEnd{c: c, style: style}
}

func (se *StartEnd) mainStartEnd() (int, int) {
	c := se.c
	switch se.style.Direction {
	case RowDirection:
		return c.Bounds.Min.X, c.Bounds.Max.X
	case ColumnDirection:
		return c.Bounds.Min.Y, c.Bounds.Max.Y
	}
	panic("!")
}
func (se *StartEnd) setMainStartEnd(s, e int) {
	c := se.c
	switch se.style.Direction {
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
func (se *StartEnd) crossStartEnd() (int, int) {
	c := se.c
	switch se.style.Direction {
	case RowDirection:
		return c.Bounds.Min.Y, c.Bounds.Max.Y
	case ColumnDirection:
		return c.Bounds.Min.X, c.Bounds.Max.X
	}
	panic("!")
}
func (se *StartEnd) setCrossStartEnd(s, e int) {
	c := se.c
	switch se.style.Direction {
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

type Style struct {
	Direction    Direction    // child containers
	Distribution Distribution // child containers

	// Individual distribution
	Hidden          bool
	MainSize        *int
	CrossSize       *int
	DynamicMainSize func() int

	// end percent distribution
	EndPercent *float64 // between 0 and 1

	// equal size distribution (no options)
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
