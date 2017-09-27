package ui

import (
	"image/color"

	"github.com/BurntSushi/xgbutil/xcursor"
	"github.com/jmigpin/editor/uiutil"
	"github.com/jmigpin/editor/xgbutil"
)

type Column struct {
	C      uiutil.Container
	Cols   *Columns
	Square *Square
	colSep *Separator
	RowsC  uiutil.Container
}

func NewColumn(cols *Columns) *Column {
	col := &Column{Cols: cols}
	col.C.Owner = col

	ui := col.Cols.Layout.UI

	col.Square = NewSquare(ui)
	col.Square.ColumnStyle = true
	height := SquareWidth
	col.Square.C.Style.CrossSize = &height
	col.Square.EvReg.Add(SquareButtonPressEventId,
		&xgbutil.ERCallback{col.onSquareButtonPress})
	col.Square.EvReg.Add(SquareButtonReleaseEventId,
		&xgbutil.ERCallback{col.onSquareButtonRelease})
	col.Square.EvReg.Add(SquareMotionNotifyEventId,
		&xgbutil.ERCallback{col.onSquareMotionNotify})

	col.colSep = NewSeparator(ui, SeparatorWidth, SeparatorColor)

	col.C.PaintFunc = col.paint
	col.RowsC.Style.Direction = uiutil.ColumnDirection
	col.RowsC.Style.Distribution = uiutil.EqualDistribution

	if ScrollbarLeft {
		col.C.AppendChilds(&col.colSep.C, &col.Square.C, &col.RowsC)
	} else {
		col.C.AppendChilds(&col.colSep.C, &col.RowsC, &col.Square.C)
	}

	return col
}
func (col *Column) Close() {
	col.Cols.removeColumn(col)
	col.Square.Close()
	for _, r := range col.Rows() {
		r.Close()
	}
}
func (col *Column) paint() {
	if col.RowsC.NChilds == 0 {
		col.Cols.Layout.UI.FillRectangle(&col.C.Bounds, color.White)
		return
	}
}

func (col *Column) NewRowBefore(next *Row) *Row {
	row := NewRow(col)
	col.insertRowBefore(row, next)
	return row
}
func (col *Column) insertRowBefore(row, next *Row) {
	row.Col = col

	var nextC *uiutil.Container
	if next != nil {
		nextC = &next.C
	}
	col.RowsC.InsertChildBefore(&row.C, nextC)

	col.fixFirstRowSeparatorAndSquare()
	col.C.CalcChildsBounds()
	col.C.NeedPaint()
}
func (col *Column) removeRow(row *Row) {
	col.RowsC.RemoveChild(&row.C)

	col.fixFirstRowSeparatorAndSquare()
	col.C.CalcChildsBounds()
	col.C.NeedPaint()
}

func (col *Column) fixFirstRowSeparatorAndSquare() {
	for i, r := range col.Rows() {
		r.HideSeparator(i == 0)
	}

	// hide/show column square if we have a first row
	_, ok := col.FirstChildRow()
	h := &col.Square.C.Style.Hidden
	haveFirstRow := ok
	if *h != haveFirstRow {
		*h = haveFirstRow
		col.C.NeedPaint()
	}
}

func (col *Column) onSquareButtonPress(ev0 interface{}) {
	ev := ev0.(*SquareButtonPressEvent)
	ui := col.Cols.Layout.UI
	switch {
	case ev.Button.Button(1):
		ui.CursorMan.SetCursor(xcursor.Fleur)
	}
}
func (col *Column) onSquareButtonRelease(ev0 interface{}) {
	ev := ev0.(*SquareButtonReleaseEvent)
	ui := col.Cols.Layout.UI
	ui.CursorMan.UnsetCursor()

	switch {
	case ev.Button.Mods.IsButton(1):
		col.Cols.MoveColumnToPoint(col, ev.Point)
	case ev.Button.Mods.IsButton(2):
		if ev.Point.In(col.Square.C.Bounds) {
			col.Cols.CloseColumnEnsureOne(col)
		}
	}
}
func (col *Column) onSquareMotionNotify(ev0 interface{}) {
	ev := ev0.(*SquareMotionNotifyEvent)
	switch {
	case ev.Mods.IsButton(3):
		p2 := ev.Point.Add(*ev.PressPointPad)
		col.Cols.resizeColumn(col, p2.X)
	}
}

func (col *Column) FirstChildRow() (*Row, bool) {
	u := col.RowsC.FirstChild
	if u == nil {
		return nil, false
	}
	return u.Owner.(*Row), true
}
func (col *Column) NextSiblingColumn() (*Column, bool) {
	u := col.C.NextSibling
	if u == nil {
		return nil, false
	}
	return u.Owner.(*Column), true
}
func (col *Column) PrevSiblingColumn() (*Column, bool) {
	u := col.C.PrevSibling
	if u == nil {
		return nil, false
	}
	return u.Owner.(*Column), true
}
func (col *Column) Rows() []*Row {
	var u []*Row
	for _, h := range col.RowsC.Childs() {
		u = append(u, h.Owner.(*Row))
	}
	return u
}

func (col *Column) HideSeparator(v bool) {
	h := &col.colSep.C.Style.Hidden
	if *h != v {
		*h = v
		col.C.NeedPaint()
	}
}
