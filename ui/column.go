package ui

import (
	"image/color"

	"github.com/jmigpin/editor/uiutil"
	"github.com/jmigpin/editor/xgbutil"
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

	ui := col.Cols.Layout.UI

	col.Square = NewSquare(ui)
	height := SquareWidth
	col.Square.C.Style.CrossSize = &height
	col.Square.EvReg.Add(SquareButtonReleaseEventId,
		&xgbutil.ERCallback{col.onSquareButtonRelease})
	col.Square.EvReg.Add(SquareMotionNotifyEventId,
		&xgbutil.ERCallback{col.onSquareMotionNotify})

	col.colSep = NewSeparator(ui, SeparatorWidth, SeparatorColor)

	col.C.PaintFunc = col.paint
	col.ChildsC.Style.Direction = uiutil.ColumnDirection
	col.ChildsC.Style.Distribution = uiutil.EqualDistribution

	col.C.AppendChilds(&col.colSep.C, &col.ChildsC, &col.Square.C)

	return col
}
func (col *Column) Close() {
	col.Cols.removeColumn(col)
	col.Square.Close()
}
func (col *Column) paint() {
	if len(col.Rows) == 0 {
		col.Cols.Layout.UI.FillRectangle(&col.C.Bounds, color.White)
		return
	}
}

func (col *Column) NewRow(index int) *Row {
	row := NewRow(col)
	col.insertRow(row, index)
	return row
}
func (col *Column) insertRow(row *Row, index int) {
	row.Col = col

	// reset separators
	row.rowSep.C.Style.Hidden = false
	if len(col.Rows) > 0 {
		col.Rows[0].rowSep.C.Style.Hidden = false
	}

	// insert
	u := make([]*Row, 0, len(col.Rows)+1)
	u = append(u, col.Rows[:index]...)
	u = append(u, row)
	u = append(u, col.Rows[index:]...)
	col.Rows = u

	// hide first row separator
	col.Rows[0].rowSep.C.Style.Hidden = true

	col.Square.C.Style.Hidden = true

	col.ChildsC.InsertChild(&row.C, index)
	col.C.CalcChildsBounds()
	col.C.NeedPaint()
}
func (col *Column) removeRow(row *Row) {
	index := col.RowIndex(row)

	// remove: ensure gargage collection
	u := make([]*Row, 0, len(col.Rows)-1)
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
func (col *Column) RowIndex(row *Row) int {
	for i, r := range col.Rows {
		if r == row {
			return i
		}
	}
	panic("row not found in this column")
}
func (col *Column) onSquareButtonRelease(ev0 xgbutil.EREvent) {
	ev := ev0.(*SquareButtonReleaseEvent)
	switch {
	case ev.Button.Mods.IsButton(1):
		col.Cols.MoveColumnToPoint(col, ev.Point)
	case ev.Button.Mods.IsButton(2):
		if ev.Point.In(col.Square.C.Bounds) {
			col.Cols.CloseColumnEnsureOne(col)
		}
	}
}
func (col *Column) onSquareMotionNotify(ev0 xgbutil.EREvent) {
	ev := ev0.(*SquareMotionNotifyEvent)
	switch {
	case ev.Mods.IsButton(3):
		p2 := ev.Point.Add(col.Square.PressPointPad)
		col.Cols.resizeColumn(col, p2.X)
	}
}
