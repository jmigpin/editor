package ui

import (
	"image"
	"image/color"

	"github.com/jmigpin/editor/uiutil"
)

type Columns struct {
	C      uiutil.Container
	Layout *Layout
	Cols   []*Column
}

func NewColumns(layout *Layout) *Columns {
	cols := &Columns{Layout: layout}
	cols.C.PaintFunc = cols.paint
	cols.C.Style.Distribution = uiutil.EndPercentDistribution

	cols.NewColumn() // ensure column
	return cols
}
func (cols *Columns) paint() {
	if len(cols.Cols) == 0 {
		cols.Layout.UI.FillRectangle(&cols.C.Bounds, color.White)
		return
	}
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
	// reset separators
	col.colSep.C.Style.Hidden = false
	if len(cols.Cols) > 0 {
		cols.Cols[0].colSep.C.Style.Hidden = false
	}

	// insert
	u := make([]*Column, 0, len(cols.Cols)+1)
	u = append(u, cols.Cols[:index]...)
	u = append(u, col)
	u = append(u, cols.Cols[index:]...)
	cols.Cols = u

	// hide first col separator
	if len(cols.Cols) > 0 {
		cols.Cols[0].colSep.C.Style.Hidden = true
	}

	cols.C.InsertChild(&col.C, index)
	cols.C.CalcChildsBounds()
	cols.C.NeedPaint()
}

// Used by restore session.
func (cols *Columns) CloseAllAndOpenN(n int) {
	// close all columns
	for len(cols.Cols) > 0 {
		cols.Cols[0].Close()
	}
	// ensure one column
	if n <= 1 {
		n = 1
	}
	// n new columns
	for ; n > 0; n-- {
		_ = cols.NewColumn()
	}
}
func (cols *Columns) CloseColumnEnsureOne(col *Column) {
	col.Close()
	// ensure one column
	if len(cols.Cols) == 0 {
		_ = cols.NewColumn()
	}
}
func (cols *Columns) removeColumn(col *Column) {
	// close all rows
	for len(col.Rows) > 0 {
		col.Rows[0].Close()
	}

	i, ok := cols.columnIndex(col)
	if !ok {
		panic("!")
	}
	// remove: new slice ensures garbage collection
	u := make([]*Column, 0, len(cols.Cols)-1)
	u = append(u, cols.Cols[:i]...)
	u = append(u, cols.Cols[i+1:]...)
	cols.Cols = u

	// hide first col separator
	if len(cols.Cols) > 0 {
		cols.Cols[0].colSep.C.Style.Hidden = true
	}

	cols.C.RemoveChild(&col.C)
	cols.C.CalcChildsBounds()
	cols.C.NeedPaint()
}
func (cols *Columns) resizeColumn(col *Column, px int) {
	ci, ok := cols.columnIndex(col)
	if !ok {
		return
	}
	colsB := col.Cols.C.Bounds
	ep := float64(px-cols.C.Bounds.Min.X) / float64(colsB.Dx())

	// minimum size
	min := float64(10) / float64(colsB.Dx())

	// limit to siblings column end percent
	if ci == 0 {
		if ep < min {
			ep = min
		}
	}
	if ci > 0 {
		u := &cols.Cols[ci-1].C.Style.EndPercent
		if *u != nil && ep < **u+min {
			ep = **u + min
		}
	}
	if ci < len(cols.Cols)-1 {
		u := &cols.Cols[ci+1].C.Style.EndPercent
		if *u != nil && ep > **u-min {
			ep = **u - min
		}
	}

	col.C.Style.EndPercent = &ep
	cols.C.CalcChildsBounds()

	//cols.C.NeedPaint() // commented: only 2 columns need paint
	col.C.NeedPaint()
	if ci < len(cols.Cols)-1 {
		cols.Cols[ci+1].C.NeedPaint()
	}
}

func (cols *Columns) columnIndex(col *Column) (int, bool) {
	for i, c := range cols.Cols {
		if c == col {
			return i, true
		}
	}
	return 0, false
}

// Row arg can be nil to allow calc before row exists.
func (cols *Columns) PointRowPosition(row *Row, p *image.Point) (*Column, int, bool) {
	for _, c := range cols.Cols {
		if !p.In(c.C.Bounds) {
			continue
		}
		if len(c.Rows) == 0 {
			return c, 0, true
		}
		for i, r := range c.Rows {
			if !p.In(r.C.Bounds) {
				continue
			}
			// don't move to itself
			if row != nil && r == row {
				return nil, 0, false
			}

			//return c, i, true

			sameCol := row != nil && row.Col == r.Col
			inFirstHalf := p.Y >= r.C.Bounds.Min.Y && p.Y < r.C.Bounds.Min.Y+r.C.Bounds.Dy()/2
			if !sameCol {
				if !inFirstHalf {
					i++
				}
			}
			return c, i, true
		}
	}
	return nil, 0, false
}
func (cols *Columns) MoveRowToColumn(row *Row, col *Column, index int) {
	row.Col.removeRow(row)
	col.insertRow(row, index)
	row.WarpPointer()
}
func (cols *Columns) MoveColumnToPoint(col *Column, p *image.Point) {
	for _, c := range cols.Cols {
		if p.In(c.C.Bounds) {
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

	ep := func(col *Column) **float64 {
		return &col.C.Style.EndPercent
	}

	bubble := func(i int) {
		a, b := &cols.Cols[i], &cols.Cols[i+1]

		// keep end percent
		start := 0.0
		if i-1 >= 0 {
			start = **ep(cols.Cols[i-1])
		}
		bep := start + (**ep(*b) - **ep(*a))
		*ep(*a), *ep(*b) = *ep(*b), &bep

		// swap at cols and at container childs
		*a, *b = *b, *a
		cols.C.SwapChilds(&(*a).C, &(*b).C)
	}

	if i0 < i1 {
		// bubble down (left)
		for i := i0; i < i1; i++ {
			bubble(i)
		}
	} else {
		// bubble up (right)
		for i := i0 - 1; i >= i1; i-- {
			bubble(i)
		}
	}

	cols.C.CalcChildsBounds()
	cols.C.NeedPaint()
	cols.Layout.UI.WarpPointerToRectanglePad(&col.C.Bounds)
}
func (cols *Columns) ColumnWithGoodPlaceForNewRow() *Column {
	var best struct {
		r    *image.Rectangle
		area int
		col  *Column
	}
	best.col = cols.Cols[0]
	rectArea := func(r *image.Rectangle) int {
		p := r.Size()
		return p.X * p.Y
	}
	for _, col := range cols.Cols {
		dy0 := col.C.Bounds.Dy()
		dy := dy0 / (len(col.Rows) + 1)

		// take into consideration the textarea content size
		usedY := 0
		for _, r := range col.Rows {
			ry := r.C.Bounds.Dy()

			// small text - count only needed height
			ry1 := ry - r.TextArea.C.Bounds.Dy()
			ry2 := ry1 + r.TextArea.StrHeight().Round()
			if ry2 < ry {
				ry = ry2
			}

			usedY += ry
		}
		dy2 := dy0 - usedY
		if dy < dy2 {
			dy = dy2
		}

		r := image.Rect(0, 0, col.C.Bounds.Dx(), dy)
		area := rectArea(&r)
		if area > best.area {
			best.area = area
			best.r = &r
			best.col = col
		}
	}
	if best.col == nil {
		panic("col is nil")
	}
	return best.col
}
