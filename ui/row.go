package ui

import (
	"github.com/BurntSushi/xgbutil/xcursor"
	"github.com/jmigpin/editor/uiutil"
	"github.com/jmigpin/editor/xgbutil/evreg"
	"github.com/jmigpin/editor/xgbutil/xinput"
)

type Row struct {
	C         uiutil.Container
	Col       *Column
	Toolbar   *Toolbar
	TextArea  *TextArea
	Square    *Square
	scrollbar *Scrollbar
	rowSep    *Separator
	EvReg     *evreg.Register
	evUnreg   evreg.Unregister

	buttonPressed bool
}

func NewRow(col *Column) *Row {
	row := &Row{Col: col}
	row.C.Owner = row
	row.EvReg = evreg.NewRegister()

	ui := row.Col.Cols.Layout.UI

	r1 := ui.EvReg.Add(xinput.KeyPressEventId,
		&evreg.Callback{row.onKeyPress})
	r2 := ui.EvReg.Add(xinput.ButtonPressEventId,
		&evreg.Callback{row.onButtonPress})
	r3 := ui.EvReg.Add(xinput.ButtonReleaseEventId,
		&evreg.Callback{row.onButtonRelease})
	row.evUnreg.Add(r1, r2, r3)

	row.Toolbar = NewToolbar(ui, &row.C)

	row.Square = NewSquare(ui)
	row.Square.EvReg.Add(SquareButtonPressEventId,
		&evreg.Callback{row.onSquareButtonPress})
	row.Square.EvReg.Add(SquareButtonReleaseEventId,
		&evreg.Callback{row.onSquareButtonRelease})
	row.Square.EvReg.Add(SquareMotionNotifyEventId,
		&evreg.Callback{row.onSquareMotionNotify})

	row.TextArea = NewTextArea(ui)
	row.TextArea.Colors = &TextAreaColors

	row.scrollbar = NewScrollbar(row.TextArea)

	// separators
	sw := SeparatorWidth
	row.rowSep = NewSeparator(ui, sw, SeparatorColor)
	tbSep := NewSeparator(ui, sw, RowInnerSeparatorColor)

	// wrap containers
	w1 := &uiutil.Container{}
	if ScrollbarLeft {
		w1.AppendChilds(&row.Square.C, &row.Toolbar.C)
	} else {
		w1.AppendChilds(&row.Toolbar.C, &row.Square.C)
	}
	w2 := &uiutil.Container{}
	if ScrollbarLeft {
		w2.AppendChilds(&row.scrollbar.C, &row.TextArea.C)
	} else {
		w2.AppendChilds(&row.TextArea.C, &row.scrollbar.C)
	}
	row.C.Style.Direction = uiutil.ColumnDirection
	row.C.AppendChilds(&row.rowSep.C, w1, &tbSep.C, w2)

	// dynamic toolbar bounds
	w1.Style.DynamicMainSize = func() int {
		dx := row.C.Bounds.Dx() - *row.Square.C.Style.MainSize
		return row.Toolbar.CalcStringHeight(dx)
	}

	return row
}
func (row *Row) activate() {
	// deactivate previous active row
	for _, c := range row.Col.Cols.Columns() {
		for _, r := range c.Rows() {
			r.Square.SetValue(SquareActive, false)
		}
	}
	// activate this row
	row.Square.SetValue(SquareActive, true)
}
func (row *Row) Close() {
	row.Col.removeRow(row)
	row.evUnreg.UnregisterAll()
	row.scrollbar.Close()
	row.Toolbar.Close()
	row.TextArea.Close()
	row.Square.Close()
	row.EvReg.RunCallbacks(RowCloseEventId, &RowCloseEvent{row})
}
func (row *Row) onSquareButtonPress(ev0 interface{}) {
	ev := ev0.(*SquareButtonPressEvent)
	ui := row.Col.Cols.Layout.UI
	switch {
	case ev.Button.Button(1):
		ui.CursorMan.SetCursor(xcursor.Fleur)
	case ev.Button.Button(2):
		ui.CursorMan.SetCursor(xcursor.XCursor)
	case ev.Button.Button(3):
		ui.CursorMan.SetCursor(xcursor.SBHDoubleArrow)
	}
}
func (row *Row) onSquareButtonRelease(ev0 interface{}) {
	ui := row.Col.Cols.Layout.UI
	ui.CursorMan.UnsetCursor()

	ev := ev0.(*SquareButtonReleaseEvent)
	switch {
	case ev.Button.Mods.IsButton(1):
		c, r, ok := row.Col.Cols.PointNextRow(row, ev.Point)
		if ok {
			row.Col.Cols.MoveRowToColumnBeforeRow(row, c, r)
		}
	case ev.Button.Mods.IsButtonAndControl(1):
		row.Col.Cols.MoveColumnToPoint(row.Col, ev.Point)
	case ev.Button.Mods.IsButton(2):
		if ev.Point.In(row.Square.C.Bounds) {
			row.Close()
		}
	}
}
func (row *Row) onSquareMotionNotify(ev0 interface{}) {
	ev := ev0.(*SquareMotionNotifyEvent)
	switch {
	case ev.Mods.IsButton(3):
		p2 := ev.Point.Add(*ev.PressPointPad)
		col := row.Col
		col.Cols.resizeColumn(col, p2.X)
	}
}
func (row *Row) onKeyPress(ev0 interface{}) {
	ev := ev0.(*xinput.KeyPressEvent)
	if !ev.Point.In(row.C.Bounds) {
		return
	}
	row.activate()
	ev2 := &RowKeyPressEvent{row, ev.Key}
	row.EvReg.RunCallbacks(RowKeyPressEventId, ev2)
}
func (row *Row) onButtonPress(ev0 interface{}) {
	ev := ev0.(*xinput.ButtonPressEvent)
	if !ev.Point.In(row.C.Bounds) {
		return
	}
	row.buttonPressed = true
}
func (row *Row) onButtonRelease(ev0 interface{}) {
	if !row.buttonPressed {
		return
	}
	row.buttonPressed = false
	ev := ev0.(*xinput.ButtonReleaseEvent)
	if !ev.Point.In(row.C.Bounds) {
		return
	}
	row.activate()
}
func (row *Row) WarpPointer() {
	row.Square.WarpPointer()
}
func (row *Row) NextSiblingRow() (*Row, bool) {
	u := row.C.NextSibling
	if u == nil {
		return nil, false
	}
	return u.Owner.(*Row), true
}

func (row *Row) HideSeparator(v bool) {
	h := &row.rowSep.C.Style.Hidden
	if *h != v {
		*h = v
		row.C.NeedPaint()
	}
}

const (
	RowKeyPressEventId = iota
	RowCloseEventId
)

type RowKeyPressEvent struct {
	Row *Row
	Key *xinput.Key
}
type RowCloseEvent struct {
	Row *Row
}
