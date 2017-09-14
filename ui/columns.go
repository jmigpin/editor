package ui

import (
	"image"
	"image/color"

	"github.com/jmigpin/editor/uiutil"
)

type Columns struct {
	C      uiutil.Container
	Layout *Layout
}

func NewColumns(layout *Layout) *Columns {
	cols := &Columns{Layout: layout}
	cols.C.PaintFunc = cols.paint
	cols.C.Style.Distribution = uiutil.EndPercentDistribution

	cols.NewColumn() // ensure column
	return cols
}
func (cols *Columns) paint() {
	if cols.C.NChilds == 0 {
		cols.Layout.UI.FillRectangle(&cols.C.Bounds, color.White)
		return
	}
}
func (cols *Columns) LastColumnOrNew() *Column {
	col, ok := cols.LastChildColumn()
	if !ok {
		col = cols.NewColumn()
	}
	return col
}
func (cols *Columns) NewColumn() *Column {
	col := NewColumn(cols)
	cols.insertColumnBefore(col, nil)
	return col
}
func (cols *Columns) insertColumnBefore(col, next *Column) {
	var nextC *uiutil.Container
	if next != nil {
		nextC = &next.C
	}
	cols.C.InsertChildBefore(&col.C, nextC)

	cols.fixFirstColSeparator()
	cols.C.CalcChildsBounds()
	cols.C.NeedPaint()
}
func (cols *Columns) removeColumn(col *Column) {
	cols.C.RemoveChild(&col.C)

	cols.fixFirstColSeparator()
	cols.C.CalcChildsBounds()
	cols.C.NeedPaint()
}

func (cols *Columns) fixFirstColSeparator() {
	for i, c := range cols.Columns() {
		c.HideSeparator(i == 0)
	}
}

// Used by restore session.
func (cols *Columns) CloseAllAndOpenN(n int) {
	// close all columns
	for cols.C.NChilds > 0 {
		u, _ := cols.FirstChildColumn()
		u.Close()
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
	if cols.C.NChilds == 0 {
		_ = cols.NewColumn()
	}
}
func (cols *Columns) resizeColumn(col *Column, px int) {
	colsB := col.Cols.C.Bounds
	ep := float64(px-cols.C.Bounds.Min.X) / float64(colsB.Dx())

	// minimum size
	min := float64(10) / float64(colsB.Dx())

	// limit to siblings column end percent
	if col.C.PrevSibling == nil {
		if ep < min {
			ep = min
		}
	}
	if col.C.PrevSibling != nil {
		u := &col.C.PrevSibling.Style.EndPercent
		if *u != nil && ep < **u+min {
			ep = **u + min
		}
	}
	if col.C.NextSibling != nil {
		u := &col.C.NextSibling.Style.EndPercent
		if *u != nil && ep > **u-min {
			ep = **u - min
		}
	}

	col.C.Style.EndPercent = &ep
	cols.C.CalcChildsBounds()

	//cols.C.NeedPaint() // commented: only 2 columns need paint
	col.C.NeedPaint()
	if col.C.NextSibling != nil {
		col.C.NextSibling.NeedPaint()
	}
}

func (cols *Columns) PointNextRow(row *Row, p *image.Point) (*Column, *Row, bool) {
	c, r := cols.pointColumnRow(row, p)
	if c == nil && r == nil {
		return nil, nil, false
	}

	// don't move to itself
	if row != nil && r == row {
		return nil, nil, false
	}

	getNext := func(r *Row) *Row {
		for u, ok := r.NextSiblingRow(); ok; u, ok = u.NextSiblingRow() {
			if u != row {
				return u
			}
		}
		return nil
	}

	if r != nil {
		sameCol := row != nil && row.Col == r.Col
		if sameCol {
			if row.C.IsAPrevSiblingOf(&r.C) {
				r = getNext(r)
			}
		} else {
			inFirstHalf := p.Y >= r.C.Bounds.Min.Y && p.Y < r.C.Bounds.Min.Y+r.C.Bounds.Dy()/2
			if !inFirstHalf {
				r = getNext(r)
			}
		}
	}

	return c, r, true
}

// Row arg can be nil to allow calc before row exists.
func (cols *Columns) pointColumnRow(row *Row, p *image.Point) (*Column, *Row) {
	for _, c := range cols.Columns() {
		if !p.In(c.C.Bounds) {
			continue
		}
		if _, ok := c.FirstChildRow(); !ok {
			return c, nil
		}
		for _, r := range c.Rows() {
			if !p.In(r.C.Bounds) {
				continue
			}
			return c, r
		}
	}
	return nil, nil
}

func (cols *Columns) MoveRowToColumnBeforeRow(row *Row, col *Column, next *Row) {
	row.Col.removeRow(row)
	col.insertRowBefore(row, next)
	row.WarpPointer()
}
func (cols *Columns) MoveColumnToPoint(col *Column, p *image.Point) {
	for _, c := range cols.Columns() {
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

	swap := func(a, b *uiutil.Container) {
		uiutil.SwapEndPercents(a, b)
		a.SwapWithSibling(b)
	}

	bubbleRight := col.C.IsAPrevSiblingOf(&dest.C)
	if bubbleRight {
		foundFirst := false
		for _, c := range col.Cols.Columns() {
			if !foundFirst {
				if c == col {
					foundFirst = true
				}
			} else {
				swap(&col.C, &c.C)
				if c == dest {
					break
				}
			}
		}
	} else {
		foundFirst := false
		u := col.Cols.Columns()
		for i, _ := range u {
			c := u[len(u)-1-i]
			if !foundFirst {
				if c == col {
					foundFirst = true
				}
			} else {
				swap(&c.C, &col.C)
				if c == dest {
					break
				}
			}
		}
	}

	cols.fixFirstColSeparator()
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

	u, ok := cols.FirstChildColumn()
	if ok {
		best.col = u
	}

	rectArea := func(r *image.Rectangle) int {
		p := r.Size()
		return p.X * p.Y
	}
	columns := cols.Columns()
	for _, col := range columns {
		dy0 := col.C.Bounds.Dy()
		dy := dy0 / (len(columns) + 1)

		// take into consideration the textarea content size
		usedY := 0
		for _, r := range col.Rows() {
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

func (cols *Columns) FirstChildColumn() (*Column, bool) {
	u := cols.C.FirstChild
	if u == nil {
		return nil, false
	}
	return u.Owner.(*Column), true
}
func (cols *Columns) LastChildColumn() (*Column, bool) {
	u := cols.C.LastChild
	if u == nil {
		return nil, false
	}
	return u.Owner.(*Column), true
}
func (cols *Columns) Columns() []*Column {
	var u []*Column
	for _, h := range cols.C.Childs() {
		u = append(u, h.Owner.(*Column))
	}
	return u
}
