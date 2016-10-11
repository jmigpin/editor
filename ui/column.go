package ui

import (
	"image"
	"image/color"
	"jmigpin/editor/xutil/dragndrop"
	"sync"
)

type Column struct {
	Container
	Cols   *Columns
	Rows   []*Row
	Square *Square
	// end percent, percentage of the columns area, determines column area size
	End float64
}

func NewColumn(cols *Columns) *Column {
	col := &Column{Cols: cols}
	col.Container.Painter = col

	col.Square = NewSquare()
	col.Square.Data = col

	col.AddChilds(&col.Square.Container)

	col.Container.OnPointEvent = col.onPointEvent
	return col
}
func (col *Column) CalcArea(area *image.Rectangle) {
	a := *area
	col.Area = a

	if col.hasSeparator() {
		a.Min.X += SeparatorWidth
	}
	// prevent square from getting events if not drawn (empty area)
	col.Square.CalcArea(&image.Rectangle{})
	if len(col.Rows) == 0 {
		// square
		r2 := a
		r2.Min.X = r2.Max.X - ScrollbarWidth
		r2.Max.Y = r2.Min.Y + ScrollbarWidth
		col.Square.CalcArea(&r2)
	}
	// each row
	var wg sync.WaitGroup
	wg.Add(len(col.Rows))
	minY := a.Min.Y
	for i, row := range col.Rows {
		// calc avoiding rounding errors
		maxY := a.Min.Y + (i+1)*a.Dy()/len(col.Rows)
		isLast := i == len(col.Rows)-1
		if isLast {
			maxY = a.Max.Y
		}
		r := image.Rect(a.Min.X, minY, a.Max.X, maxY)
		go func(row *Row, r image.Rectangle) {
			defer wg.Done()
			row.CalcArea(&r)
		}(row, r)
		minY = maxY
	}
	wg.Wait()
}
func (col *Column) Paint() {
	a := col.Area
	// separator
	if col.hasSeparator() {
		r := a
		r.Max.X = r.Min.X + SeparatorWidth
		col.FillRectangle(&r, &SeparatorColor)
		a.Min.X = r.Max.X
	}
	// square
	if len(col.Rows) == 0 {
		// background when empty
		col.FillRectangle(&a, color.White)

		col.Square.Paint()
	}
	// each row
	var wg sync.WaitGroup
	wg.Add(len(col.Rows))
	for _, row := range col.Rows {
		go func(row *Row) {
			defer wg.Done()
			row.Paint()
		}(row)
	}
	wg.Wait()
}
func (col *Column) hasSeparator() bool {
	index, ok := col.Cols.columnIndex(col)
	if !ok {
		panic("column not found")
	}
	// separator is on the left side
	return index > 0
}
func (col *Column) NewRow() *Row {
	row := NewRow(col)
	col.insertRow(row, len(col.Rows))
	return row
}
func (col *Column) insertRow(row *Row, index int) {
	row.Col = col

	// insert: ensure gargage collection
	var u []*Row
	u = append(u, col.Rows[:index]...)
	u = append(u, row)
	u = append(u, col.Rows[index:]...)
	col.Rows = u

	col.AddChilds(&row.Container)
	col.CalcOwnArea()
	col.NeedPaint()
}
func (col *Column) removeRow(row *Row) {
	index, ok := col.rowIndex(row)
	if !ok {
		panic("row doesn't belong to col")
	}

	// remove: ensure gargage collection
	var u []*Row
	u = append(u, col.Rows[:index]...)
	u = append(u, col.Rows[index+1:]...)
	col.Rows = u

	col.RemoveChild(&row.Container)
	col.CalcOwnArea()
	col.NeedPaint()
}
func (col *Column) rowIndex(row *Row) (int, bool) {
	for i, r := range col.Rows {
		if r == row {
			return i, true
		}
	}
	return 0, false
}
func (col *Column) onPointEvent(p *image.Point, ev Event) bool {
	switch ev0 := ev.(type) {
	case *dragndrop.PositionEvent:
		col.UI.PushEvent(&ColumnDndPositionEvent{ev0, p, col})
	case *dragndrop.DropEvent:
		col.UI.PushEvent(&ColumnDndDropEvent{ev0, p, col})
	}
	return true
}
