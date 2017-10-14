package ui

import (
	"image"
	"image/color"

	"github.com/BurntSushi/xgbutil/xcursor"
	"github.com/jmigpin/editor/uiutil/widget"
)

type Column struct {
	widget.FlowLayout
	Square     *Square
	sep        *widget.Space
	rowsLayout *widget.EndPercentLayout

	sqc *widget.FlowLayout // square container (show/hide)

	Cols *Columns

	resize struct {
		on     bool
		origin image.Point
	}
}

func NewColumn(cols *Columns) *Column {
	col := &Column{Cols: cols}

	ui := col.Cols.Layout.UI

	col.Square = NewSquare(ui)
	col.Square.EvReg.Add(SquareButtonPressEventId, col.onSquareButtonPress)
	col.Square.EvReg.Add(SquareButtonReleaseEventId, col.onSquareButtonRelease)
	col.Square.EvReg.Add(SquareMotionNotifyEventId, col.onSquareMotionNotify)

	col.sep = widget.NewSpace(ui)
	col.sep.SetExpand(false, true)
	col.sep.Size.X = SeparatorWidth
	col.sep.Color = SeparatorColor

	col.rowsLayout = &widget.EndPercentLayout{YAxis: true}

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
	widget.AppendChilds(rightSide, col.sqc, col.rowsLayout)

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
	if len(col.rowsLayout.Childs()) == 0 {
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
	case ev.Button.Button(2):
		// indicate close
		ui.CursorMan.SetCursor(xcursor.XCursor)
	case ev.Button.Button(3):
		ui.CursorMan.SetCursor(xcursor.SBHDoubleArrow)
		col.startResizeToPoint(ev.Point)
	}
}
func (col *Column) onSquareMotionNotify(ev0 interface{}) {
	ev := ev0.(*SquareMotionNotifyEvent)
	switch {
	case ev.Mods.IsButton(3):
		col.resizeToPoint(ev.Point)
	}
}
func (col *Column) onSquareButtonRelease(ev0 interface{}) {
	ev := ev0.(*SquareButtonReleaseEvent)

	ui := col.Cols.Layout.UI
	ui.CursorMan.UnsetCursor()

	switch {
	case ev.Button.Mods.IsButton(2):
		if ev.Point.In(col.Square.Bounds()) {
			col.Cols.CloseColumnEnsureOne(col)
		}
	case ev.Button.Mods.IsButton(3):
		col.endResizeToPoint(ev.Point)
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

func (col *Column) PointRow(p *image.Point) (*Row, bool) {
	for _, r := range col.Rows() {
		if p.In(r.Bounds()) {
			return r, true
		}
	}
	return nil, false
}

func (col *Column) startResizeToPoint(p *image.Point) {
	col.resize.on = true
	col.resize.origin = p.Sub(col.Square.Bounds().Min)
}
func (col *Column) resizeToPoint(p *image.Point) {
	if col.resize.on {
		col.resizeToPointOrigin(p, &col.resize.origin)
	}
}
func (col *Column) endResizeToPoint(p *image.Point) {
	if col.resize.on {
		col.resizeToPointOrigin(p, &col.resize.origin)
	}
	col.resize.on = false
}

func (col *Column) resizeToPointOrigin(p *image.Point, origin *image.Point) {
	bounds := col.Cols.Layout.Bounds()
	dx := float64(bounds.Dx())
	perc := float64(p.Sub(*origin).Sub(bounds.Min).X) / dx
	min := 30 / dx

	if !ScrollbarLeft {
		u, ok := col.NextColumn()
		if ok {
			col = u
		}
	}

	col.Cols.ResizeEndPercent(col, perc, min)
	col.Cols.AttemptToSwap(col.Cols, col, perc, min)

	col.Cols.fixFirstColSeparator()
	col.Cols.CalcChildsBounds()
	col.Cols.MarkNeedsPaint()
}
