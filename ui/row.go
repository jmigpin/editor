package ui

import (
	"image"

	"github.com/jmigpin/editor/uiutil"
	"github.com/jmigpin/editor/xutil/keybmap"
	"github.com/jmigpin/editor/xutil/xgbutil"
)

type Row struct {
	C             uiutil.Container
	Col           *Column
	Toolbar       *Toolbar
	TextArea      *TextArea
	Square        *Square
	scrollbar     *Scrollbar
	rowSep        *Separator
	EvReg         *xgbutil.EventRegister
	dereg         xgbutil.EventDeregister
	buttonPressed bool
}

func NewRow(col *Column) *Row {
	row := &Row{Col: col}
	row.EvReg = xgbutil.NewEventRegister()

	ui := row.Col.Cols.Layout.UI

	r1 := ui.Win.EvReg.Add(keybmap.KeyPressEventId,
		&xgbutil.ERCallback{row.onKeyPress})
	r2 := ui.Win.EvReg.Add(keybmap.ButtonPressEventId,
		&xgbutil.ERCallback{row.onButtonPress})
	r3 := ui.Win.EvReg.Add(keybmap.ButtonReleaseEventId,
		&xgbutil.ERCallback{row.onButtonRelease})
	row.dereg.Add(r1, r2, r3)

	row.Toolbar = NewToolbar(ui)
	tb := row.Toolbar
	tb.Colors = &ToolbarColors

	row.Square = NewSquare(ui)
	row.Square.EvReg.Add(SquareButtonReleaseEventId,
		&xgbutil.ERCallback{row.onSquareButtonRelease})
	row.Square.EvReg.Add(SquareMotionNotifyEventId,
		&xgbutil.ERCallback{row.onSquareMotionNotify})

	row.TextArea = NewTextArea(ui)
	row.TextArea.Colors = &TextAreaColors

	row.scrollbar = NewScrollbar(row.TextArea)

	// separators
	sw := SeparatorWidth
	row.rowSep = NewSeparator(ui, sw, SeparatorColor)
	tbSep := NewSeparator(ui, sw, RowInnerSeparatorColor)

	// wrap containers
	w1 := &uiutil.Container{}
	w1.AppendChilds(&row.Toolbar.C, &row.Square.C)
	w2 := &uiutil.Container{}
	w2.AppendChilds(&row.TextArea.C, &row.scrollbar.C)
	row.C.Style.Direction = uiutil.ColumnDirection
	row.C.AppendChilds(&row.rowSep.C, w1, &tbSep.C, w2)

	// dynamic toolbar bounds
	tb.OnSetText = func() {
		b := tb.C.Bounds
		row.C.CalcChildsBounds()
		if !tb.C.Bounds.Eq(b) {
			row.C.NeedPaint()
		}
	}
	w1.Style.DynamicMainSize = func() int {
		dx := row.C.Bounds.Dx() - *row.Square.C.Style.MainSize
		return row.Toolbar.CalcStringHeight(dx)
	}

	return row
}
func (row *Row) activate() {
	// deactivate previous active row
	for _, c := range row.Col.Cols.Cols {
		for _, r := range c.Rows {
			r.Square.SetValue(SquareActive, false)
		}
	}
	// activate this row
	row.Square.SetValue(SquareActive, true)
}
func (row *Row) Close() {
	row.Col.removeRow(row)
	row.dereg.UnregisterAll()
	row.scrollbar.Close()
	row.Toolbar.Close()
	row.TextArea.Close()
	row.Square.Close()
	row.EvReg.Emit(RowCloseEventId, &RowCloseEvent{row})
}
func (row *Row) onSquareButtonRelease(ev0 xgbutil.EREvent) {
	ev := ev0.(*SquareButtonReleaseEvent)
	switch {
	case ev.Button.Mods.IsButton(1):
		c, i, ok := row.Col.Cols.PointRowPosition(row, ev.Point)
		if ok {
			row.Col.Cols.MoveRowToColumn(row, c, i)
		}
	case ev.Button.Mods.IsButtonAndControl(1):
		row.Col.Cols.MoveColumnToPoint(row.Col, ev.Point)
	case ev.Button.Mods.IsButton(2):
		if ev.Point.In(row.Square.C.Bounds) {
			row.Close()
		}
	}
}
func (row *Row) onSquareMotionNotify(ev0 xgbutil.EREvent) {
	ev := ev0.(*SquareMotionNotifyEvent)
	switch {
	case ev.Mods.IsButton(3):
		p2 := ev.Point.Add(row.Square.PressPointPad)
		col := row.Col
		col.Cols.resizeColumn(col, p2.X)
	}
}
func (row *Row) onKeyPress(ev0 xgbutil.EREvent) {
	ev := ev0.(*keybmap.KeyPressEvent)
	if !ev.Point.In(row.C.Bounds) {
		return
	}
	row.activate()
	ev2 := &RowKeyPressEvent{row, ev.Key}
	row.EvReg.Emit(RowKeyPressEventId, ev2)
}
func (row *Row) onButtonPress(ev0 xgbutil.EREvent) {
	ev := ev0.(*keybmap.ButtonPressEvent)
	if !ev.Point.In(row.C.Bounds) {
		return
	}
	row.buttonPressed = true
}
func (row *Row) onButtonRelease(ev0 xgbutil.EREvent) {
	if !row.buttonPressed {
		return
	}
	row.buttonPressed = false
	ev := ev0.(*keybmap.ButtonReleaseEvent)
	if !ev.Point.In(row.C.Bounds) {
		return
	}
	row.activate()
}
func (row *Row) WarpPointer() {
	b := row.C.Bounds
	p := b.Min.Add(image.Pt(b.Dx()/2, b.Dy()/3))
	row.Col.Cols.Layout.UI.WarpPointer(&p)
}

const (
	RowKeyPressEventId = iota
	RowCloseEventId
)

type RowKeyPressEvent struct {
	Row *Row
	Key *keybmap.Key
}
type RowCloseEvent struct {
	Row *Row
}
