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

	//cols.NewColumn() // ensure column
	return cols
}
func (cols *Columns) paint() {
	if len(cols.Cols) == 0 {
		cols.Layout.UI.FillRectangle(&cols.C.Bounds, color.White)
		return
	}
}

//func (cols *Columns) CalcArea(area *image.Rectangle) {
//cols.C.Bounds = *area
//// each column
//minX := cols.C.Bounds.Min.X
//var wg sync.WaitGroup
//wg.Add(len(cols.Cols))
//for i, col := range cols.Cols {
//maxX := cols.C.Bounds.Min.X + int(col.End*float64(cols.C.Bounds.Dx()))
//isLast := i == len(cols.Cols)-1
//if isLast {
//maxX = cols.C.Bounds.Max.X
//}
//r := image.Rect(minX, cols.C.Bounds.Min.Y, maxX, cols.C.Bounds.Max.Y)
//go func(col *Column, r image.Rectangle) {
//defer wg.Done()
//col.CalcArea(&r)
//}(col, r)
//minX = maxX
//}
//wg.Wait()
//}
//func (cols *Columns) Paint() {
//// background when empty
//if len(cols.Cols) == 0 {
//cols.FillRectangle(&cols.C.Bounds, color.Black)
//return
//}
//// each column
//var wg sync.WaitGroup
//wg.Add(len(cols.Cols))
//for _, col := range cols.Cols {
//go func(col *Column) {
//col.Paint()
//wg.Done()
//}(col)
//}
//wg.Wait()
//}
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
	// show col separator
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

	cols.C.AppendChilds(&col.C)
	//cols.calcColumnEndBasedOnSiblings(col)
	cols.C.CalcChildsBounds()
	cols.C.NeedPaint()
}

//func (cols *Columns) calcColumnEndBasedOnSiblings(col *Column) {
//i, ok := cols.columnIndex(col)
//if !ok {
//panic("column not found")
//}

//cend := func(col *Column) float64 {
//if col.C.Style.EndPercent == nil {
//return 5 // small number to debug
//}
//return *col.C.Style.EndPercent
//}
//setcend := func(col *Column, v float64) {
//col.C.Style.EndPercent = &v
//}

//if i == 0 {
//if i == len(cols.Cols)-1 {
//setcend(col, 100)
//} else {
//next := cols.Cols[i+1]
//setcend(col, cend(next)/2)
//}
//} else if i == len(cols.Cols)-1 {
//ppe := 0.0
//if i >= 2 {
//ppe = cend(cols.Cols[i-2])
//}
//prev := cols.Cols[i-1]
//// only case where another column end is changed
//setcend(prev, ppe+(cend(prev)-ppe)/2)
//setcend(col, 100)
//} else {
//prev := cols.Cols[i-1]
//next := cols.Cols[i+1]
//setcend(col, cend(prev)+(cend(next)-cend(prev))/2)
//}
//}
func (cols *Columns) RemoveColumnEnsureOne(col *Column) {
	cols.RemoveColumn(col)
	// ensure at least one column
	if len(cols.Cols) == 0 {
		_ = cols.NewColumn()
	}
}
func (cols *Columns) RemoveColumn(col *Column) {
	// close all rows
	for _, row := range col.Rows {
		row.Close()
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
	// limit to siblings column end percent
	if ci > 0 {
		u := &cols.Cols[ci-1].C.Style.EndPercent
		if *u != nil && ep < **u {
			ep = **u
		}
	}
	if ci < len(cols.Cols)-1 {
		u := &cols.Cols[ci+1].C.Style.EndPercent
		if *u != nil && ep > **u {
			ep = **u
		}
	}

	col.C.Style.EndPercent = &ep
	cols.C.CalcChildsBounds()
	cols.C.NeedPaint()

	//i, ok := cols.columnIndex(col)
	//if !ok {
	//return
	//}

	//endPos := float64(px - cols.C.Bounds.Min.X)
	//col.End = endPos / float64(cols.C.Bounds.Dx())

	//// check limits
	//minWidth := ScrollbarWidth * 2
	//min := float64(minWidth) / float64(cols.C.Bounds.Dx())
	//pe, ne := 0.0, 1.0 // previous/next end
	//if i-1 >= 0 {
	//pe = cols.Cols[i-1].End
	//}
	//if i+1 < len(cols.Cols) {
	//ne = cols.Cols[i+1].End
	//}
	//if col.End < pe+min {
	//col.End = pe + min
	//}
	//if col.End > ne-min {
	//col.End = ne - min
	//}

	//// at most 2 colums need paint
	//col.C.NeedPaint()
	//if i+1 < len(cols.Cols) {
	//next := cols.Cols[i+1]
	//next.C.NeedPaint()
	//}
	//// the two column endpercents are calculated, but the areas themselfs are calculated from columns, and all need to be redone
	//col.Cols.C.CalcChildsBounds()
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

			sameCol := row != nil && row.Col == r.Col
			inFirstHalf := p.Y >= r.C.Bounds.Min.Y && p.Y < r.C.Bounds.Min.Y+r.C.Bounds.Dy()/2

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
	cols.Layout.UI.WarpPointerToRectangle(&row.C.Bounds)
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
	//if col == dest {
	//return
	//}
	//i0, ok := cols.columnIndex(col)
	//if !ok {
	//return
	//}
	//i1, ok := cols.columnIndex(dest)
	//if !ok {
	//return
	//}

	//bubble := func(i int) {
	//a, b := &cols.Cols[i], &cols.Cols[i+1]

	//// keep end percent
	//start := 0.0
	//if i-1 >= 0 {
	//start = cols.Cols[i-1].End
	//}
	//bep := start + ((*b).End - (*a).End)
	//(*a).End, (*b).End = (*b).End, bep

	////// get destination end percent
	////(*a).End, (*b).End = (*b).End, (*a).End

	//*a, *b = *b, *a
	//}
	//if i0 < i1 {
	//// bubble down
	//for i := i0; i < i1; i++ {
	//bubble(i)
	//}
	//} else {
	//// bubble up
	//for i := i0 - 1; i >= i1; i-- {
	//bubble(i)
	//}
	//}

	//cols.C.CalcChildsBounds()
	//cols.C.NeedPaint()
	//cols.Layout.UI.WarpPointerToRectangle(&col.C.Bounds)
}
