package ui

import (
	"image"

	"github.com/jmigpin/editor/imageutil"
	"github.com/jmigpin/editor/uiutil/event"
	"github.com/jmigpin/editor/uiutil/widget"
)

type Column struct {
	*widget.FlowLayout
	Square     *Square
	Cols       *Columns
	RowsLayout *widget.EndPercentLayout

	sep           widget.Rectangle
	sqc           *widget.FlowLayout // square container (show/hide)
	closingCursor bool
}

func NewColumn(cols *Columns) *Column {
	col := &Column{Cols: cols}
	col.FlowLayout = widget.NewFlowLayout()
	col.SetWrapper(col)

	ui := col.Cols.Layout.UI

	col.Square = NewSquare(ui)
	col.Square.EvReg.Add(SquareInputEventId, col.onSquareInput)

	col.sep.Init(ui)
	col.sep.SetExpand(false, true)
	col.sep.Size.X = SeparatorWidth
	col.sep.Color = &SeparatorColor

	col.RowsLayout = widget.NewEndPercentLayout()
	col.RowsLayout.YAxis = true

	// square (when there are no rows)
	col.sqc = widget.NewFlowLayout()
	var sqBorder widget.Pad
	sqBorder.Init(ui, col.Square)
	sqBorder.Color = &RowInnerSeparatorColor
	sqBorder.Bottom = SeparatorWidth
	var space widget.Rectangle
	space.Init(ui)
	space.SetFill(true, true)
	if ScrollbarLeft {
		sqBorder.Right = SeparatorWidth
		col.sqc.Append(&sqBorder, &space)
	} else {
		sqBorder.Left = SeparatorWidth
		col.sqc.Append(&space, &sqBorder)
	}

	rightSide := widget.NewFlowLayout()
	rightSide.YAxis = true
	rightSide.Append(col.sqc, col.RowsLayout)

	col.Append(&col.sep, rightSide)

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
		ui := col.Cols.Layout.UI
		b := col.Bounds()
		imageutil.FillRectangle(ui.Image(), &b, ColumnBgColor)
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
		col.RowsLayout.PushBack(row)
	} else {
		col.RowsLayout.InsertBefore(row, next)
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
			ui.SetCursor(widget.CloseCursor)
		}

	case *event.MouseClick:
		switch ev.Button {
		case event.ButtonMiddle:
			col.Cols.CloseColumnEnsureOne(col)
			ui.SetCursor(widget.NoCursor)
		}

	case *event.MouseDragStart:
		if col.closingCursor {
			col.closingCursor = false
			ui.SetCursor(widget.NoCursor)
		}
		switch ev.Button {
		case event.ButtonLeft, event.ButtonRight:
			ui.SetCursor(widget.WEResizeCursor)
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
			ui.SetCursor(widget.NoCursor)
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
	col.Cols.ResizeEndPercentWithSwap(col, perc, percIsLeft, min)

	col.Cols.fixFirstColSeparator()
	col.Cols.CalcChildsBounds()
	col.Cols.MarkNeedsPaint()
}
