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
}

func NewRow(col *Column) *Row {
	row := &Row{Col: col}

	row.Toolbar = NewToolbar(col.Cols.Layout.UI)
	tb := row.Toolbar
	tb.Data = row
	tb.Colors = &ToolbarColors
	tb.DisableButtonScroll = true
	tb.OnSetText = func() {
		// dynamic toolbar bounds
		row.C.CalcChildsBounds()
		row.C.NeedPaint() // TODO: if bounds changed
	}

	row.TextArea = NewTextArea(col.Cols.Layout.UI)
	row.TextArea.Data = row
	row.TextArea.Colors = &TextAreaColors
	fn := &xgbutil.ERCallback{row.onTextAreaScroll}
	row.TextArea.EvReg.Add(TextAreaScrollEventId, fn)
	fn = &xgbutil.ERCallback{row.onTextAreaSetOffsetY}
	row.TextArea.EvReg.Add(TextAreaSetOffsetYEventId, fn)

	row.Square = NewSquare(row.Col.Cols.Layout.UI)
	fn = &xgbutil.ERCallback{row.onSquareButtonRelease}
	row.Square.EvReg.Add(SquareButtonReleaseEventId, fn)
	fn = &xgbutil.ERCallback{row.onSquareMotionNotify}
	row.Square.EvReg.Add(SquareMotionNotifyEventId, fn)

	row.scrollbar = NewScrollbar(row.TextArea)

	ui := row.Col.Cols.Layout.UI
	row.rowSep = NewSeparator(ui, SeparatorWidth, SeparatorColor)
	tbSep := NewSeparator(ui, SeparatorWidth, RowInnerSeparatorColor)

	row.C.Style.Direction = uiutil.ColumnDirection
	w1 := &uiutil.Container{}
	w1.Style.DynamicMainSize = func() int {
		// FIXME: bounds don't include the square, so text width won't be correct
		row.Toolbar.C.Bounds = row.C.Bounds
		return row.Toolbar.CalcUsedY()
	}
	w1.AppendChilds(&row.Toolbar.C, &row.Square.C)
	w2 := &uiutil.Container{}
	w2.AppendChilds(&row.TextArea.C, &row.scrollbar.C)
	row.C.AppendChilds(&row.rowSep.C, w1, &tbSep.C, w2)

	row.EvReg = xgbutil.NewEventRegister()

	fn = &xgbutil.ERCallback{row.onKeyPress}
	row.Col.Cols.Layout.UI.Win.EvReg.Add(keybmap.KeyPressEventId, fn)

	return row
}

//func (row *Row) CalcArea(area *image.Rectangle) {

//a := *area
//row.C.Bounds = a
//// separator
//if row.hasSeparator() {
//a.Min.Y += SeparatorWidth
//}
//// toolbar
//r1 := a
//r1.Max.X -= ScrollbarWidth
//r1 = r1.Intersect(a)
//row.Toolbar.CalcArea(&r1)
//// square
//r2 := a
//r2.Min.X = r2.Max.X - ScrollbarWidth
//r2.Max.Y = row.Toolbar.C.Bounds.Max.Y
//r2 = r2.Intersect(a)
//row.Square.CalcArea(&r2)
//// horizontal separator
//r5 := a
//r5.Min.Y = r2.Max.Y + 1
//a = r5.Intersect(a)
//// textarea
//r3 := a
//r3.Max.X -= ScrollbarWidth
//r3 = r3.Intersect(a)
//row.TextArea.CalcArea(&r3)
//// scrollbar
//r4 := a
//r4.Min.X = r4.Max.X - ScrollbarWidth
//r4 = r4.Intersect(a)
//row.scrollbar.CalcArea(&r4)
//}

//func (row *Row) Paint() {
//// separator
//if row.hasSeparator() {
//r := row.C.Bounds
//r.Max.Y = r.Min.Y + SeparatorWidth
//row.FillRectangle(&r, &SeparatorColor)
//}
//row.Toolbar.Paint()
//row.Square.Paint()

//// horizontal separator
//r3 := row.C.Bounds
//r3.Min.Y = row.Toolbar.C.Bounds.Max.Y
//r3.Max.Y = r3.Min.Y + 1
//r3 = r3.Intersect(row.C.Bounds)
//row.FillRectangle(&r3, &RowInnerSeparatorColor)

//row.TextArea.Paint()
//row.scrollbar.Paint()
//}
//func (row *Row) hasSeparator() bool {
//index, ok := row.Col.rowIndex(row)
//if !ok {
//panic("!")
//}
//// separator is on the top
//return index > 0
//}

//func (row *Row) onPointEvent(p *image.Point, ev Event) bool {
//switch ev0 := ev.(type) {
//case *KeyPressEvent:
//ev2 := &RowKeyPressEvent{row, ev0.Key}
//row.UI.PushEvent(ev2)
//case *ButtonPressEvent:
//row.activate()
//}
//return true
//}
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
	ev := &RowCloseEvent{row}
	row.EvReg.Emit(RowCloseEventId, ev)
}
func (row *Row) onTextAreaScroll(ev0 xgbutil.EREvent) {
	//ev := ev0.(*TextAreaScrollEvent)
	row.scrollbar.C.NeedPaint()
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
func (row *Row) onTextAreaSetOffsetY(ev0 xgbutil.EREvent) {
	//ev:=ev0.(*TextAreaSetOffsetYEvent)
	row.scrollbar.C.CalcChildsBounds()
	row.scrollbar.C.NeedPaint()
}
func (row *Row) onKeyPress(ev0 xgbutil.EREvent) {
	ev := ev0.(*keybmap.KeyPressEvent)
	if ev.Point.In(row.C.Bounds) {
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
