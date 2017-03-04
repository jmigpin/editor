package ui

import (
	"image"
	"image/color"
	"sync"
)

type Columns struct {
	Container
	layout *Layout
	Cols   []*Column
}

func NewColumns(layout *Layout) *Columns {
	cols := &Columns{layout: layout}
	cols.Container.Painter = cols
	return cols
}
func (cols *Columns) CalcArea(area *image.Rectangle) {
	cols.Area = *area
	// each column
	minX := cols.Area.Min.X
	var wg sync.WaitGroup
	wg.Add(len(cols.Cols))
	for i, col := range cols.Cols {
		maxX := cols.Area.Min.X + int(col.End*float64(cols.Area.Dx()))
		isLast := i == len(cols.Cols)-1
		if isLast {
			maxX = cols.Area.Max.X
		}
		r := image.Rect(minX, cols.Area.Min.Y, maxX, cols.Area.Max.Y)
		go func(col *Column, r image.Rectangle) {
			defer wg.Done()
			col.CalcArea(&r)
		}(col, r)
		minX = maxX
	}
	wg.Wait()
}
func (cols *Columns) Paint() {
	// background when empty
	if len(cols.Cols) == 0 {
		cols.FillRectangle(&cols.Area, color.Black)
		return
	}
	// each column
	var wg sync.WaitGroup
	wg.Add(len(cols.Cols))
	for _, col := range cols.Cols {
		go func(col *Column) {
			col.Paint()
			wg.Done()
		}(col)
	}
	wg.Wait()
}
func (cols *Columns) LastColumnOrNew() *Column {
	if len(cols.Cols) == 0 {
		col := cols.NewColumn()
		return col
	}
	return cols.Cols[len(cols.Cols)-1]
}
func (cols *Columns) NewColumn() *Column {
	col := NewColumn(cols)
	cols.insertColumn(col, len(cols.Cols))
	return col
}
func (cols *Columns) insertColumn(col *Column, index int) {
	// insert
	u := make([]*Column, 0, len(cols.Cols)+1)
	u = append(u, cols.Cols[:index]...)
	u = append(u, col)
	u = append(u, cols.Cols[index:]...)
	cols.Cols = u

	cols.calcColumnEndBasedOnNeighbours(col)

	cols.AddChilds(&col.Container)
	cols.CalcOwnArea()
	cols.NeedPaint()
}
func (cols *Columns) calcColumnEndBasedOnNeighbours(col *Column) {
	i, ok := cols.columnIndex(col)
	if !ok {
		panic("column not found")
	}
	if i == 0 {
		if i == len(cols.Cols)-1 {
			col.End = 1.0
		} else {
			next := cols.Cols[i+1]
			col.End = next.End / 2
		}
	} else if i == len(cols.Cols)-1 {
		ppe := 0.0
		if i >= 2 {
			ppe = cols.Cols[i-2].End
		}
		prev := cols.Cols[i-1]
		// only case where another column end is changed
		prev.End = ppe + (prev.End-ppe)/2
		col.End = 1.0
	} else {
		prev := cols.Cols[i-1]
		next := cols.Cols[i+1]
		col.End = prev.End + (next.End-prev.End)/2
	}
}
func (cols *Columns) RemoveColumnEnsureOne(col *Column) {
	cols.RemoveColumnUntilNone(col)
	// ensure at least one column
	if len(cols.Cols) == 0 {
		_ = cols.NewColumn()
	}
}
func (cols *Columns) RemoveColumnUntilNone(col *Column) {
	// close all rows
	for _, row := range col.Rows {
		row.Close()
	}

	i, ok := cols.columnIndex(col)
	if !ok {
		return
	}
	// remove: new slice ensures garbage collection
	u := make([]*Column, 0, len(cols.Cols)-1)
	u = append(u, cols.Cols[:i]...)
	u = append(u, cols.Cols[i+1:]...)
	cols.Cols = u
	// removing a column doesn't touch the end percents, just ensure the last one
	if len(cols.Cols) > 0 {
		cols.Cols[len(cols.Cols)-1].End = 1.0
	}
	cols.RemoveChild(&col.Container)
	cols.CalcOwnArea()
	cols.NeedPaint()
}
func (cols *Columns) resizeColumn(col *Column, px int) {
	i, ok := cols.columnIndex(col)
	if !ok {
		return
	}

	endPos := float64(px - cols.Area.Min.X)
	col.End = endPos / float64(cols.Area.Dx())

	// check limits
	minWidth := ScrollbarWidth * 2
	min := float64(minWidth) / float64(cols.Area.Dx())
	pe, ne := 0.0, 1.0 // previous/next end
	if i-1 >= 0 {
		pe = cols.Cols[i-1].End
	}
	if i+1 < len(cols.Cols) {
		ne = cols.Cols[i+1].End
	}
	if col.End < pe+min {
		col.End = pe + min
	}
	if col.End > ne-min {
		col.End = ne - min
	}

	// at most 2 colums need paint
	col.NeedPaint()
	if i+1 < len(cols.Cols) {
		next := cols.Cols[i+1]
		next.NeedPaint()
	}
	// the two column endpercents are calculated, but the areas themselfs are calculated from columns, and all need to be redone
	col.Cols.CalcOwnArea()
}

func (cols *Columns) columnIndex(col *Column) (int, bool) {
	for i, c := range cols.Cols {
		if c == col {
			return i, true
		}
	}
	return 0, false
}

// Row arg can be nil.
func (cols *Columns) PointRowPosition(row *Row, p *image.Point) (*Column, int, bool) {
	for _, c := range cols.Cols {
		if !p.In(c.Area) {
			continue
		}
		if len(c.Rows) == 0 {
			return c, 0, true
		}
		for i, r := range c.Rows {
			if !p.In(r.Area) {
				continue
			}
			// don't move to itself
			if row != nil && r == row {
				return nil, 0, false
			}

			sameCol := row != nil && row.Col == r.Col
			inFirstHalf := p.Y >= r.Area.Min.Y && p.Y < r.Area.Min.Y+r.Area.Dy()/2

			index := i
			if !sameCol {
				if !inFirstHalf {
					index++
				}
			}
			return c, index, true
		}
	}
	return nil, 0, false
}
func (cols *Columns) MoveRowToColumn(row *Row, col *Column, index int) {
	row.Col.removeRow(row)
	col.insertRow(row, index)
	cols.UI.WarpPointerToRectangle(&row.Area)
}
func (cols *Columns) MoveColumnToPoint(col *Column, p *image.Point) {
	for _, c := range cols.Cols {
		if p.In(c.Area) {
			cols.moveColumnToColumn(col, c, p)
			break
		}
	}
}
func (cols *Columns) moveColumnToColumn(col, dest *Column, p *image.Point) {
	if col == dest {
		return
	}
	i0, ok := cols.columnIndex(col)
	if !ok {
		return
	}
	i1, ok := cols.columnIndex(dest)
	if !ok {
		return
	}

	bubble := func(i int) {
		a, b := &cols.Cols[i], &cols.Cols[i+1]

		// keep end percent
		start := 0.0
		if i-1 >= 0 {
			start = cols.Cols[i-1].End
		}
		bep := start + ((*b).End - (*a).End)
		(*a).End, (*b).End = (*b).End, bep

		//// get destination end percent
		//(*a).End, (*b).End = (*b).End, (*a).End

		*a, *b = *b, *a
	}
	if i0 < i1 {
		// bubble down
		for i := i0; i < i1; i++ {
			bubble(i)
		}
	} else {
		// bubble up
		for i := i0 - 1; i >= i1; i-- {
			bubble(i)
		}
	}

	cols.CalcOwnArea()
	cols.NeedPaint()
	cols.UI.WarpPointerToRectangle(&col.Area)
}
