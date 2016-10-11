package ui

import "image"

type Row struct {
	Container
	Col *Column

	Toolbar   *TextArea
	TextArea  *TextArea
	Square    *Square
	scrollbar *Scrollbar
}

func NewRow(col *Column) *Row {
	row := &Row{Col: col}
	row.Container.Painter = row

	row.Toolbar = NewTextArea()
	row.Toolbar.Data = row
	row.Toolbar.DynamicY = true
	row.Toolbar.Colors = &ToolbarColors

	row.TextArea = NewTextArea()
	row.TextArea.Data = row
	row.TextArea.Colors = &TextAreaColors

	row.Square = NewSquare()
	row.Square.Data = row

	row.scrollbar = NewScrollbar(row.TextArea)

	row.Container.OnPointEvent = row.onPointEvent

	row.AddChilds(
		&row.Toolbar.Container,
		&row.Square.Container,
		&row.TextArea.Container,
		&row.scrollbar.Container)

	return row
}
func (row *Row) CalcArea(area *image.Rectangle) {
	a := *area
	row.Area = a
	// separator
	if row.hasSeparator() {
		a.Min.Y += SeparatorWidth
	}
	// toolbar
	r1 := a
	r1.Max.X -= ScrollbarWidth
	r1 = r1.Intersect(a)
	row.Toolbar.CalcArea(&r1)
	// square
	r2 := a
	r2.Min.X = r2.Max.X - ScrollbarWidth
	r2.Max.Y = row.Toolbar.Area.Max.Y
	r2 = r2.Intersect(a)
	row.Square.CalcArea(&r2)
	// horizontal separator
	a.Min.Y = row.Toolbar.Area.Max.Y + 1
	// textarea
	r3 := a
	r3.Max.X -= ScrollbarWidth
	r3 = r3.Intersect(a)
	row.TextArea.CalcArea(&r3)
	// scrollbar
	r4 := a
	r4.Min.X = r4.Max.X - ScrollbarWidth
	r4 = r4.Intersect(a)
	row.scrollbar.CalcArea(&r4)
}
func (row *Row) Paint() {
	// separator
	if row.hasSeparator() {
		r := row.Area
		r.Max.Y = r.Min.Y + SeparatorWidth
		row.FillRectangle(&r, &SeparatorColor)
	}
	row.Toolbar.Paint()
	row.Square.Paint()

	// horizontal separator
	r3 := row.Area
	r3.Min.Y = row.Toolbar.Area.Max.Y
	r3.Max.Y = r3.Min.Y + 1
	r3 = r3.Intersect(row.Area)
	row.FillRectangle(&r3, &RowInnerSeparatorColor)

	row.TextArea.Paint()
	row.scrollbar.Paint()
}
func (row *Row) hasSeparator() bool {
	index, ok := row.Col.rowIndex(row)
	if !ok {
		panic("!")
	}
	// separator is on the top
	return index > 0
}
func (row *Row) onPointEvent(p *image.Point, ev Event) bool {
	switch ev0 := ev.(type) {
	case *KeyPressEvent:
		ev2 := &RowKeyPressEvent{row, ev0.Key}
		row.UI.PushEvent(ev2)
	case *ButtonPressEvent:
		row.activate()
	}
	return true
}
func (row *Row) activate() {
	// deactivate previous active row
	for _, c := range row.Col.Cols.Cols {
		for _, r := range c.Rows {
			r.Square.SetActive(false)
		}
	}
	// activate this row
	row.Square.SetActive(true)
}
func (row *Row) Close() {
	row.Col.removeRow(row)
	row.UI.PushEvent(&RowCloseEvent{row})
}
