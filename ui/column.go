package ui

import (
	"image/color"

	"github.com/jmigpin/editor/uiutil"
	"github.com/jmigpin/editor/xutil/xgbutil"
)

type Column struct {
	C, ChildsC uiutil.Container
	Cols       *Columns
	Rows       []*Row
	Square     *Square
	colSep     *Separator
}

func NewColumn(cols *Columns) *Column {
	col := &Column{Cols: cols}

	col.Square = NewSquare(cols.Layout.UI)
	fn := &xgbutil.ERCallback{col.onSquareButtonRelease}
	col.Square.EvReg.Add(SquareButtonReleaseEventId, fn)
	fn = &xgbutil.ERCallback{col.onSquareMotionNotify}
	col.Square.EvReg.Add(SquareMotionNotifyEventId, fn)

	ui := col.Cols.Layout.UI
	col.colSep = NewSeparator(ui, SeparatorWidth, SeparatorColor)

	col.C.PaintFunc = col.paint
	col.ChildsC.Style.Direction = uiutil.ColumnDirection
	col.ChildsC.Style.Distribution = uiutil.EqualDistribution

	col.C.AppendChilds(&col.colSep.C, &col.ChildsC, &col.Square.C)

	return col
}
func (col *Column) paint() {
	if len(col.Rows) == 0 {
		col.Cols.Layout.UI.FillRectangle(&col.C.Bounds, color.White)
		return
	}
}
func (col *Column) NewRow() *Row {
	row := NewRow(col)
	col.insertRow(row, len(col.Rows))
	return row
}
func (col *Column) insertRow(row *Row, index int) {
	row.Col = col

	// show first row separator
	if len(col.Rows) > 0 {
		col.Rows[0].rowSep.C.Style.Hidden = false
	}

	// insert: ensure gargage collection
	var u []*Row
	u = append(u, col.Rows[:index]...)
	u = append(u, row)
	u = append(u, col.Rows[index:]...)
	col.Rows = u

	// hide first row separator
	col.Rows[0].rowSep.C.Style.Hidden = true

	col.Square.C.Style.Hidden = true

	col.ChildsC.AppendChilds(&row.C)
	col.C.CalcChildsBounds()
	col.C.NeedPaint()
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

	// hide first row separator
	if len(col.Rows) > 0 {
		col.Rows[0].rowSep.C.Style.Hidden = true
	}

	col.Square.C.Style.Hidden = len(col.Rows) > 0

	col.ChildsC.RemoveChild(&row.C)
	col.C.CalcChildsBounds()
	col.C.NeedPaint()
}
func (col *Column) rowIndex(row *Row) (int, bool) {
	for i, r := range col.Rows {
		if r == row {
			return i, true
		}
	}
	return 0, false
}
func (col *Column) onSquareButtonRelease(ev0 xgbutil.EREvent) {
	ev := ev0.(*SquareButtonReleaseEvent)
	switch {
	case ev.Button.Button1():
		col.Cols.MoveColumnToPoint(col, ev.Point)
	case ev.Button.Button2():
		if ev.Point.In(col.Square.C.Bounds) {
			col.Cols.RemoveColumnEnsureOne(col)
		}
	}
}
func (col *Column) onSquareMotionNotify(ev0 xgbutil.EREvent) {
	ev := ev0.(*SquareMotionNotifyEvent)
	switch {
	case ev.Modifiers.Button3():
		p2 := ev.Point.Add(col.Square.PressPointPad)
		col.Cols.resizeColumn(col, p2.X)
	}
}
