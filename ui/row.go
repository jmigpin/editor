package ui

import (
	"github.com/jmigpin/editor/uiutil"
	"github.com/jmigpin/editor/xutil/keybmap"
	"github.com/jmigpin/editor/xutil/xgbutil"
)

type Row struct {
	C         uiutil.Container
	Col       *Column
	Toolbar   *Toolbar
	TextArea  *TextArea
	Square    *Square
	scrollbar *Scrollbar
	rowSep    *Separator
	EvReg     *xgbutil.EventRegister
	dereg     xgbutil.EventDeregister
}

func NewRow(col *Column) *Row {
	row := &Row{Col: col}
	row.EvReg = xgbutil.NewEventRegister()

	ui := row.Col.Cols.Layout.UI

	// capture keypress to emit rowkeypress for key shortcuts
	r1 := ui.Win.EvReg.Add(keybmap.KeyPressEventId,
		&xgbutil.ERCallback{row.onKeyPress})
	row.dereg.Add(r1)

	row.Toolbar = NewToolbar(ui)
	tb := row.Toolbar
	tb.Colors = &ToolbarColors
	tb.DisableButtonScroll = true

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
			r.Square.SetActive(false)
		}
	}
	// activate this row
	row.Square.SetActive(true)
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
	case ev.Button.Button1():
		col := row.Col
		if ev.Button.Mods.Control() {
			col.Cols.MoveColumnToPoint(col, ev.Point)
		} else {
			c, i, ok := col.Cols.PointRowPosition(row, ev.Point)
			if ok {
				col.Cols.MoveRowToColumn(row, c, i)
			}
		}
	case ev.Button.Button2():
		if ev.Point.In(row.Square.C.Bounds) {
			row.Close()
		}
	}
}
func (row *Row) onSquareMotionNotify(ev0 xgbutil.EREvent) {
	ev := ev0.(*SquareMotionNotifyEvent)
	switch {
	case ev.Modifiers.Button3():
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
	ev2 := &RowKeyPressEvent{row, ev.Key}
	row.EvReg.Emit(RowKeyPressEventId, ev2)
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
