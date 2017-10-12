package ui

import (
	"image"
	"image/color"

	"github.com/jmigpin/editor/uiutil/widget"
)

type Columns struct {
	widget.EndPercentLayout
	Layout *Layout
}

func NewColumns(layout *Layout) *Columns {
	cols := &Columns{Layout: layout}

	cols.NewColumn() // start with 1 column

	return cols
}
func (cols *Columns) Paint() {
	if cols.NChilds() == 0 {
		b := cols.Bounds()
		cols.Layout.UI.FillRectangle(&b, color.White)
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
	cols.InsertBefore(col, nil)
	return col
}
func (cols *Columns) InsertBefore(col, next *Column) {
	if next == nil {

		// TODO: need to return false

		//// don't insert if it will be too small
		//lc := cols.LastChild()
		//if lc != nil && lc.Prev() != nil {
		//	start := cols.ChildEndPercent(lc.Prev())
		//	end := cols.ChildEndPercent(lc)
		//	x := int((end - start) * float64(cols.Bounds().Dx()))
		//	if x < 40 {
		//		return
		//	}
		//}

		widget.PushBack(cols, col)
	} else {
		panic("TODO")
		widget.InsertBefore(cols, col, next)
	}

	cols.fixFirstColSeparator()
	cols.CalcChildsBounds()
	cols.MarkNeedsPaint()
}

// TODO: override node.Remove?

func (cols *Columns) removeColumn(col *Column) {
	cols.Remove(col)
	cols.fixFirstColSeparator()
	cols.CalcChildsBounds()
	cols.MarkNeedsPaint()
}

func (cols *Columns) fixFirstColSeparator() {
	for i, c := range cols.Columns() {
		c.HideSeparator(i == 0)
	}
}

func (cols *Columns) CloseColumnEnsureOne(col *Column) {
	col.Close()
	// ensure one column
	if cols.NChilds() == 0 {
		_ = cols.NewColumn()
	}
}

// Used by restore session.
func (cols *Columns) CloseAllAndOpenN(n int) {
	// close all columns
	for cols.NChilds() > 0 {
		u, _ := cols.FirstChildColumn()
		u.Close()
	}
	// ensure one column
	if n <= 1 {
		n = 1
	}
	// n new columns
	for i := 0; i < n; i++ {
		_ = cols.NewColumn()
	}
}

func (cols *Columns) resizeColumn(col *Column, px int) {
	if ScrollbarLeft {
		u, ok := col.PrevColumn()
		if !ok {
			return
		}
		col = u
	}

	bounds := cols.Bounds()
	ep := float64(px-bounds.Min.X) / float64(bounds.Dx())

	// minimum size
	min := float64(10) / float64(bounds.Dx())

	// limit to siblings column end percent
	if col.Prev() == nil {
		if ep < min {
			ep = min
		}
	}
	if col.Prev() != nil {
		u := cols.ChildEndPercent(col.Prev())
		if ep < u+min {
			ep = u + min
		}
	}
	if col.Next() != nil {
		u := cols.ChildEndPercent(col.Next())
		if ep > u-min {
			ep = u - min
		}
	}

	cols.SetChildEndPercent(col, ep)
	cols.CalcChildsBounds()

	// only 2 columns need paint
	col.MarkNeedsPaint()
	if col.Next() != nil {
		col.Next().MarkNeedsPaint()
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
		for u, ok := r.NextRow(); ok; u, ok = u.NextRow() {
			if u != row {
				return u
			}
		}
		return nil
	}

	if r != nil {
		sameCol := row != nil && row.Col == r.Col
		if sameCol {
			if widget.IsAPrevOf(row, r) {
				r = getNext(r)
			}
		} else {
			inFirstHalf := p.Y >= r.Bounds().Min.Y && p.Y < r.Bounds().Min.Y+r.Bounds().Dy()/2
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
		if !p.In(c.Bounds()) {
			continue
		}
		if _, ok := c.FirstChildRow(); !ok {
			return c, nil
		}
		for _, r := range c.Rows() {
			if !p.In(r.Bounds()) {
				continue
			}
			return c, r
		}
	}
	return nil, nil
}

func (cols *Columns) MoveRowToColumnBeforeRow(row *Row, col *Column, next *Row) {
	row.Col.removeRow(row)
	col.insertBefore(row, next)
	row.WarpPointer()
}

func (cols *Columns) MoveColumnToPoint(col *Column, p *image.Point) {
	for _, c := range cols.Columns() {
		if p.In(c.Bounds()) {
			cols.moveColumnToColumn(col, c, p)
			break
		}
	}
}
func (cols *Columns) moveColumnToColumn(col, dest *Column, p *image.Point) {
	if col == dest {
		return
	}

	col.Swap(dest)
	a1 := cols.ChildEndPercent(col)
	a2 := cols.ChildEndPercent(dest)
	cols.SetChildEndPercent(col, a2)
	cols.SetChildEndPercent(dest, a1)

	cols.fixFirstColSeparator()
	cols.CalcChildsBounds()
	cols.MarkNeedsPaint()
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
		dy0 := col.Bounds().Dy()
		dy := dy0 / (len(columns) + 1)

		// take into consideration the textarea content size
		usedY := 0
		for _, r := range col.Rows() {
			ry := r.Bounds().Dy()

			// small text - count only needed height
			ry1 := ry - r.TextArea.Bounds().Dy()
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

		r := image.Rect(0, 0, col.Bounds().Dx(), dy)
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
	u := cols.FirstChild()
	if u == nil {
		return nil, false
	}
	return u.(*Column), true
}
func (cols *Columns) LastChildColumn() (*Column, bool) {
	u := cols.LastChild()
	if u == nil {
		return nil, false
	}
	return u.(*Column), true
}
func (cols *Columns) Columns() []*Column {
	childs := cols.Childs()
	u := make([]*Column, 0, len(childs))
	for _, h := range childs {
		u = append(u, h.(*Column))
	}
	return u
}
