package ui

import (
	"image"
	"image/color"

	"github.com/BurntSushi/xgbutil/xcursor"
	"github.com/jmigpin/editor/uiutil/event"
	"github.com/jmigpin/editor/uiutil/widget"
)

type Column struct {
	widget.FlowLayout
	Square     *Square
	Cols       *Columns
	RowsLayout *widget.EndPercentLayout

	sep           *widget.Space
	sqc           *widget.FlowLayout // square container (show/hide)
	closingCursor bool
}

func NewColumn(cols *Columns) *Column {
	col := &Column{Cols: cols}

	ui := col.Cols.Layout.UI

	col.Square = NewSquare(ui)
	col.Square.EvReg.Add(SquareInputEventId, col.onSquareInput)

	col.sep = widget.NewSpace(ui)
	col.sep.SetExpand(false, true)
	col.sep.Size.X = SeparatorWidth
	col.sep.Color = SeparatorColor

	col.RowsLayout = &widget.EndPercentLayout{YAxis: true}

	// square (when there are no rows)
	col.sqc = &widget.FlowLayout{}
	sqBorder := widget.NewBorder(ui, col.Square)
	sqBorder.Color = RowInnerSeparatorColor
	sqBorder.Bottom = SeparatorWidth
	sep1 := widget.NewSpace(ui)
	sep1.Color = RowInnerSeparatorColor
	sep1.Size = image.Point{SeparatorWidth, col.Square.Width}
	space := widget.NewSpace(ui)
	space.SetFill(true, true)
	space.Color = nil // filled by full bg paint
	if ScrollbarLeft {
		widget.AppendChilds(col.sqc, sqBorder, sep1, space)
	} else {
		widget.AppendChilds(col.sqc, space, sep1, sqBorder)
	}

	rightSide := &widget.FlowLayout{YAxis: true}
	widget.AppendChilds(rightSide, col.sqc, col.RowsLayout)

	widget.AppendChilds(col, col.sep, rightSide)

	return col
}
func (col *Column) Close() {
	col.Cols.removeColumn(col)
	for _, r := range col.Rows() {
		r.Close()
	}
}
func (col *Column) Paint() {
	if len(col.RowsLayout.Childs()) == 0 {
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
		widget.PushBack(col.RowsLayout, row)
	} else {
		widget.InsertBefore(col.RowsLayout, row, next)
	}
	col.CalcChildsBounds()
	col.MarkNeedsPaint()
}

func (col *Column) removeRow(row *Row) {
	col.RowsLayout.Remove(row)
	col.CalcChildsBounds()
	col.MarkNeedsPaint()
}

func (col *Column) CalcChildsBounds() {
	col.fixFirstRowSeparatorAndSquare()
	col.FlowLayout.CalcChildsBounds()
}

func (col *Column) fixFirstRowSeparatorAndSquare() {
	// hide/show column square if we have a first row
	_, ok := col.FirstChildRow()
	hide := ok
	if col.sqc.Hidden() != hide {
		col.sqc.SetHidden(hide)
		col.MarkNeedsPaint()
	}
	// hide first row separator
	for i, r := range col.Rows() {
		r.HideSeparator(i == 0)
	}
}

func (col *Column) onSquareInput(ev0 interface{}) {
	sqEv := ev0.(*SquareInputEvent)
	ui := col.Cols.Layout.UI
	switch ev := sqEv.Event.(type) {
	case *event.MouseDown:
		switch ev.Button {
		case event.ButtonMiddle:
			col.closingCursor = true
			ui.CursorMan.SetCursor(xcursor.XCursor)
		}

	case *event.MouseClick:
		switch ev.Button {
		case event.ButtonMiddle:
			col.Cols.CloseColumnEnsureOne(col)
			ui.CursorMan.UnsetCursor()
		}

	case *event.MouseDragStart:
		if col.closingCursor {
			col.closingCursor = false
			ui.CursorMan.UnsetCursor()
		}
		switch ev.Button {
		case event.ButtonLeft, event.ButtonRight:
			ui.CursorMan.SetCursor(xcursor.SBHDoubleArrow)
			col.resizeToPoint(sqEv.TopPoint)
		}
	case *event.MouseDragMove:
		switch {
		case ev.Buttons.Has(event.ButtonLeft) || ev.Buttons.Has(event.ButtonRight):
			col.resizeToPoint(sqEv.TopPoint)
		}
	case *event.MouseDragEnd:
		switch ev.Button {
		case event.ButtonLeft, event.ButtonRight:
			col.resizeToPoint(sqEv.TopPoint)
			ui.CursorMan.UnsetCursor()
		}
	}
}

func (col *Column) FirstChildRow() (*Row, bool) {
	u := col.RowsLayout.FirstChild()
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
	childs := col.RowsLayout.Childs()
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

func (col *Column) PointRow(p *image.Point) (*Row, bool) {
	for _, r := range col.Rows() {
		if p.In(r.Bounds()) {
			return r, true
		}
	}
	return nil, false
}

func (col *Column) resizeToPoint(p *image.Point) {
	bounds := col.Cols.Layout.Bounds()
	dx := float64(bounds.Dx())
	perc := float64(p.Sub(bounds.Min).X) / dx
	min := 30 / dx

	percIsLeft := ScrollbarLeft
	col.Cols.ResizeEndPercentWithSwap(col.Cols, col, perc, percIsLeft, min)

	col.Cols.fixFirstColSeparator()
	col.Cols.CalcChildsBounds()
	col.Cols.MarkNeedsPaint()
}
