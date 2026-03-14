package ui

import "image"

func goodRowPos(ui *UI) *RowPos {
	p, ok := pointerPoint(ui)
	if !ok {
		return goodRowPosLargestArea(ui)
	}
	if pos, ok := preferSideRowPosFromPointer(ui, p); ok {
		return pos
	}
	return bestRowPosFromPointer(ui, p)
}

func preferSideRowPosFromPointer(ui *UI, p image.Point) (*RowPos, bool) {
	curCol, ok := ui.Root.Cols.PointColumnExtra(&p)
	if !ok || curCol == nil {
		return nil, false
	}

	for _, col := range sideColumnsByPointerDistance(curCol, p) {
		c := candidateAtPointerHeight(ui, col, p)
		if c == nil {
			continue
		}
		if hasEnoughVisibleAreaForSideOpen(ui, c, curCol) {
			return c.pos, true
		}
	}
	return nil, false
}

func bestRowPosFromPointer(ui *UI, p image.Point) *RowPos {
	cands := rowPosCandidates(ui)
	var best *rowPosCandidate
	for _, c := range cands {
		if c == nil {
			continue
		}
		scoreCandidateForPointer(ui, c, p)
		if best == nil || c.score > best.score {
			best = c
		}
	}
	if best == nil {
		return goodRowPosLargestArea(ui)
	}
	return best.pos
}

func pointerPoint(ui *UI) (image.Point, bool) {
	p, err := ui.QueryPointer()
	if err == nil {
		return p, true
	}
	return image.Point{}, false
}

func goodRowPosLargestArea(ui *UI) *RowPos {
	var best struct {
		area    int
		col     *Column
		nextRow *Row
	}

	best.col = ui.Root.Cols.FirstChildColumn()

	for _, c := range ui.Root.Cols.Columns() {
		rows := c.Rows()

		s := c.Bounds.Size()
		if len(rows) > 0 {
			s.Y = rows[0].Bounds.Min.Y - c.Bounds.Min.Y
		}
		a := s.X * s.Y
		if a > best.area {
			best.area = a
			best.col = c
			best.nextRow = nil
			if len(rows) > 0 {
				best.nextRow = rows[0]
			}
		}

		for _, r := range rows {
			b := ui.rowInsertionBounds(r)
			s := b.Size()
			a := s.X * s.Y
			if a > best.area {
				best.area = a
				best.col = c
				best.nextRow = r.NextRow()
			}
		}
	}

	return NewRowPos(best.col, best.nextRow)
}

type rowPosCandidate struct {
	pos   *RowPos
	rect  image.Rectangle
	score int
}

func rowPosCandidates(ui *UI) []*rowPosCandidate {
	cands := []*rowPosCandidate{}
	for _, col := range ui.Root.Cols.Columns() {
		cands = append(cands, columnRowPosCandidates(ui, col)...)
	}
	return cands
}

func columnRowPosCandidates(ui *UI, col *Column) []*rowPosCandidate {
	rows := col.Rows()
	cands := []*rowPosCandidate{}

	if len(rows) == 0 {
		return append(cands, &rowPosCandidate{
			pos:  NewRowPos(col, nil),
			rect: col.Bounds,
		})
	}

	if c := candidateBeforeFirstRow(col, rows[0]); c != nil {
		cands = append(cands, c)
	}
	for _, row := range rows {
		if c := candidateBelowRow(ui, row); c != nil {
			cands = append(cands, c)
		}
	}
	return cands
}

func candidateBeforeFirstRow(col *Column, first *Row) *rowPosCandidate {
	r := col.Bounds
	r.Max.Y = first.Bounds.Min.Y
	if r.Empty() {
		return nil
	}
	return &rowPosCandidate{
		pos:  NewRowPos(col, first),
		rect: r,
	}
}

func candidateBelowRow(ui *UI, row *Row) *rowPosCandidate {
	rect := ui.rowInsertionBounds(row)
	if rect.Empty() {
		return nil
	}
	return &rowPosCandidate{
		pos:  row.PosBelow(),
		rect: rect,
	}
}

func sideColumnsByPointerDistance(curCol *Column, p image.Point) []*Column {
	cs := []*Column{}
	if c := prevColumn(curCol); c != nil {
		cs = append(cs, c)
	}
	if c := nextColumn(curCol); c != nil {
		cs = append(cs, c)
	}
	if len(cs) == 2 {
		d0 := distanceToRect(p, cs[0].Bounds)
		d1 := distanceToRect(p, cs[1].Bounds)
		if d1 < d0 {
			cs[0], cs[1] = cs[1], cs[0]
		}
	}
	return cs
}

func prevColumn(col *Column) *Column {
	u := col.PrevSiblingWrapper()
	if u == nil {
		return nil
	}
	return u.(*Column)
}

func nextColumn(col *Column) *Column {
	u := col.NextSiblingWrapper()
	if u == nil {
		return nil
	}
	return u.(*Column)
}

func pointerLineHeight(ui *UI) int {
	for _, c := range ui.Root.Cols.Columns() {
		for _, r := range c.Rows() {
			if r.HasState(RowStateActive) {
				return r.TextArea.LineHeight()
			}
		}
	}
	return 16
}

func candidateAtPointerHeight(ui *UI, col *Column, p image.Point) *rowPosCandidate {
	if col == nil {
		return nil
	}
	p2 := p
	p2.X = clampInt(p.X, col.Bounds.Min.X+1, col.Bounds.Max.X-1)
	p2.Y = clampInt(p.Y, col.Bounds.Min.Y+1, col.Bounds.Max.Y-1)
	next, ok := col.PointNextRowExtra(&p2)
	if !ok {
		return nil
	}
	rect := rowPosInsertionBounds(ui, col, next)
	if rect.Empty() {
		return nil
	}
	return &rowPosCandidate{
		pos:  NewRowPos(col, next),
		rect: rect,
	}
}

func rowPosInsertionBounds(ui *UI, col *Column, next *Row) image.Rectangle {
	switch {
	case next == nil:
		last := col.LastChildRow()
		if last == nil {
			return col.Bounds
		}
		return ui.rowInsertionBounds(last)
	default:
		if u := next.PrevSiblingWrapper(); u != nil {
			return ui.rowInsertionBounds(u.(*Row))
		}
		b := col.Bounds
		b.Max.Y = next.Bounds.Min.Y
		return b
	}
}

func hasEnoughVisibleAreaForSideOpen(ui *UI, c *rowPosCandidate, curCol *Column) bool {
	return c.rect.Dy() >= sideOpenMinHeight(ui) &&
		c.rect.Dx() >= sideOpenMinWidth(curCol)
}

func sideOpenMinHeight(ui *UI) int {
	return pointerLineHeight(ui) * 5
}

func sideOpenMinWidth(curCol *Column) int {
	return curCol.Bounds.Dx() / 2
}

func scoreCandidateForPointer(ui *UI, c *rowPosCandidate, p image.Point) {
	area := c.rect.Dx() * c.rect.Dy()
	score := area
	score -= pointerDistancePenalty(c, p)
	score -= pointerOverlapPenalty(ui, c, p)
	score -= crampedAreaPenalty(ui, c)
	c.score = score
}

func pointerDistancePenalty(c *rowPosCandidate, p image.Point) int {
	return distanceToRect(p, c.rect) * maxInt(20, c.rect.Dx()/3)
}

func pointerOverlapPenalty(ui *UI, c *rowPosCandidate, p image.Point) int {
	lh := pointerLineHeight(ui)
	area := c.rect.Dx() * c.rect.Dy()
	dist := distanceToRect(p, c.rect)
	if p.In(c.rect) {
		return area * 3
	}
	if dist < lh*3 {
		return (lh*3 - dist) * maxInt(40, c.rect.Dx()/2)
	}
	return 0
}

func crampedAreaPenalty(ui *UI, c *rowPosCandidate) int {
	lh := pointerLineHeight(ui)
	penalty := 0
	if c.rect.Dy() < lh*5 {
		penalty += (lh*5 - c.rect.Dy()) * c.rect.Dx() * 4
	}
	colW := ui.Root.Bounds.Dx() / maxInt(1, len(ui.Root.Cols.Columns()))
	if c.rect.Dx() < colW/2 {
		penalty += (colW/2 - c.rect.Dx()) * lh * 4
	}
	return penalty
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampInt(v, min, max int) int {
	if min > max {
		return min
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func distanceToRect(p image.Point, r image.Rectangle) int {
	x := clampInt(p.X, r.Min.X, r.Max.X)
	y := clampInt(p.Y, r.Min.Y, r.Max.Y)
	return absInt(p.X-x) + absInt(p.Y-y)
}
