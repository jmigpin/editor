package ui

import (
	"image/color"

	"github.com/BurntSushi/xgbutil/xcursor"
	"github.com/jmigpin/editor/uiutil/widget"
	"github.com/jmigpin/editor/xgbutil/evreg"
)

type Column struct {
	widget.FlowLayout
	Square     *Square
	sep        *widget.Space
	rowsLayout *widget.FlowLayout

	sqc *widget.FlowLayout // square container (show/hide)

	Cols *Columns
}

func NewColumn(cols *Columns) *Column {
	col := &Column{Cols: cols}

	ui := col.Cols.Layout.UI

	col.Square = NewSquare(ui)
	col.Square.ColumnStyle = true
	col.Square.EvReg.Add(SquareButtonPressEventId,
		&evreg.Callback{col.onSquareButtonPress})
	col.Square.EvReg.Add(SquareButtonReleaseEventId,
		&evreg.Callback{col.onSquareButtonRelease})
	col.Square.EvReg.Add(SquareMotionNotifyEventId,
		&evreg.Callback{col.onSquareMotionNotify})

	col.sep = widget.NewSpace(ui)
	col.sep.SetExpand(false, true)
	col.sep.Size.X = SeparatorWidth
	col.sep.Color = SeparatorColor

	col.rowsLayout = &widget.FlowLayout{YAxis: true}

	// square (when there are no rows)
	col.sqc = &widget.FlowLayout{}
	space := widget.NewSpace(ui)
	space.SetFill(true, true)
	space.Color = nil
	if ScrollbarLeft {
		widget.AppendChilds(col.sqc, col.Square, space)
	} else {
		widget.AppendChilds(col.sqc, space, col.Square)
	}

	rightSide := &widget.FlowLayout{YAxis: true}
	widget.AppendChilds(rightSide, col.sqc, col.rowsLayout)

	widget.AppendChilds(col, col.sep, rightSide)

	return col
}
func (col *Column) Close() {
	col.Cols.removeColumn(col)
	col.Square.Close()
	for _, r := range col.Rows() {
		r.Close()
	}
}
func (col *Column) Paint() {
	if col.rowsLayout.NChilds() == 0 {
		b := col.Bounds()
		col.Cols.Layout.UI.FillRectangle(&b, color.White)
		return
	}
}

func (col *Column) NewRowBefore(next *Row) *Row {
	row := NewRow(col)
	col.insertBefore(row, next)
	return row
}

func (col *Column) insertBefore(row, next *Row) {
	row.Col = col
	if next == nil {
		widget.PushBack(col.rowsLayout, row)
	} else {
		widget.InsertBefore(col.rowsLayout, row, next)
	}
	col.fixFirstRowSeparatorAndSquare()
	col.CalcChildsBounds()
	col.MarkNeedsPaint()
}

func (col *Column) removeRow(row *Row) {
	col.rowsLayout.Remove(row)
	col.fixFirstRowSeparatorAndSquare()
	col.CalcChildsBounds()
	col.MarkNeedsPaint()
}

func (col *Column) fixFirstRowSeparatorAndSquare() {
	for i, r := range col.Rows() {
		r.HideSeparator(i == 0)
	}

	// hide/show column square if we have a first row
	_, ok := col.FirstChildRow()
	hide := ok
	if col.sqc.Hidden() != hide {
		col.sqc.SetHidden(hide)
		col.MarkNeedsPaint()
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
		if ev.Point.In(col.Square.Bounds()) {
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
	u := col.rowsLayout.FirstChild()
	if u == nil {
		return nil, false
	}
	return u.(*Row), true
}
func (col *Column) NextColumn() (*Column, bool) {
	u := col.Next()
	if u == nil {
		return nil, false
	}
	return u.(*Column), true
}
func (col *Column) PrevColumn() (*Column, bool) {
	u := col.Prev()
	if u == nil {
		return nil, false
	}
	return u.(*Column), true
}
func (col *Column) Rows() []*Row {
	childs := col.rowsLayout.Childs()
	u := make([]*Row, 0, len(childs))
	for _, h := range childs {
		u = append(u, h.(*Row))
	}
	return u
}

func (col *Column) HideSeparator(v bool) {
	if col.sep.Hidden() != v {
		col.sep.SetHidden(v)
		col.MarkNeedsPaint()
	}
}
