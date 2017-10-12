package ui

import (
	"github.com/BurntSushi/xgbutil/xcursor"
	"github.com/jmigpin/editor/uiutil/widget"
	"github.com/jmigpin/editor/xgbutil/evreg"
	"github.com/jmigpin/editor/xgbutil/xinput"
)

type Row struct {
	widget.FlowLayout
	Square     *Square
	Toolbar    *Toolbar
	scrollArea *ScrollArea
	TextArea   *TextArea
	sep        *widget.Space

	Col     *Column
	EvReg   *evreg.Register
	evUnreg evreg.Unregister

	buttonPressed bool

	state RowState
}

func NewRow(col *Column) *Row {
	row := &Row{Col: col}

	ui := row.Col.Cols.Layout.UI

	row.EvReg = evreg.NewRegister()
	r1 := ui.EvReg.Add(xinput.KeyPressEventId, row.onKeyPress)
	r2 := ui.EvReg.Add(xinput.ButtonPressEventId, row.onButtonPress)
	r3 := ui.EvReg.Add(xinput.ButtonReleaseEventId, row.onButtonRelease)
	row.evUnreg.Add(r1, r2, r3)

	row.Toolbar = NewToolbar(ui, row)
	row.Toolbar.SetExpand(true, false)

	row.Square = NewSquare(ui)
	row.Square.SetFill(false, true)
	row.Square.EvReg.Add(SquareButtonPressEventId, row.onSquareButtonPress)
	row.Square.EvReg.Add(SquareButtonReleaseEventId, row.onSquareButtonRelease)
	row.Square.EvReg.Add(SquareMotionNotifyEventId, row.onSquareMotionNotify)

	// row separator from other rows
	row.sep = widget.NewSpace(ui)
	row.sep.SetExpand(true, false)
	row.sep.Size.Y = SeparatorWidth
	row.sep.Color = SeparatorColor

	// square and toolbar
	tb := &widget.FlowLayout{}
	sep1 := widget.NewSpace(ui)
	sep1.Color = RowInnerSeparatorColor
	sep1.Size.X = SeparatorWidth
	sep1.SetFill(false, true)
	if ScrollbarLeft {
		widget.AppendChilds(tb, row.Square, sep1, row.Toolbar)
	} else {
		widget.AppendChilds(tb, row.Toolbar, sep1, row.Square)
	}

	// toolbar separator from scrollarea
	tbSep := widget.NewSpace(ui)
	tbSep.SetExpand(true, false)
	tbSep.Size.Y = SeparatorWidth
	tbSep.Color = RowInnerSeparatorColor

	// scrollarea with textarea
	row.TextArea = NewTextArea(ui)
	row.TextArea.Colors = &TextAreaColors
	row.scrollArea = NewScrollArea(ui, row.TextArea)
	row.scrollArea.SetExpand(true, true)
	row.scrollArea.LeftScroll = ScrollbarLeft
	row.scrollArea.ScrollWidth = ScrollbarWidth
	row.scrollArea.Fg = ScrollbarFgColor
	row.scrollArea.Bg = ScrollbarBgColor

	row.YAxis = true
	widget.AppendChilds(row, row.sep, tb, tbSep, row.scrollArea)

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
	row.scrollArea.Close()
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
		// toggle maximize
		r, ok := row.Col.Cols.PointRow(ev.Point)
		if ok && r == row {
			row.toggleMaximize()
			break
		}

		// TODO: review
		c, r, ok := row.Col.Cols.PointNextRow(row, ev.Point)
		if ok {

			// fix row states on this column before moving the row
			if row.state == RowMaximizedState {
				row.setRowsState(RowNormalState)
			}
			row.setState(RowNormalState)

			row.Col.Cols.MoveRowToColumnBeforeRow(row, c, r)
		}
	case ev.Button.Mods.IsButtonAndControl(1):
		row.Col.Cols.MoveColumnToPoint(row.Col, ev.Point)
	case ev.Button.Mods.IsButton(2):
		if ev.Point.In(row.Square.Bounds()) {
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
	if !ev.Point.In(row.Bounds()) {
		return
	}
	row.activate()
	ev2 := &RowKeyPressEvent{row, ev.Key}
	row.EvReg.RunCallbacks(RowKeyPressEventId, ev2)
}
func (row *Row) onButtonPress(ev0 interface{}) {
	ev := ev0.(*xinput.ButtonPressEvent)
	if !ev.Point.In(row.Bounds()) {
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
	if !ev.Point.In(row.Bounds()) {
		return
	}
	row.activate()
}
func (row *Row) WarpPointer() {
	row.Square.WarpPointer()
}

func (row *Row) NextRow() (*Row, bool) {
	u := row.Next()
	if u == nil {
		return nil, false
	}
	return u.(*Row), true
}

func (row *Row) HideSeparator(v bool) {
	if row.sep.Hidden() != v {
		row.sep.SetHidden(v)
		row.MarkNeedsPaint()
	}
}

func (row *Row) toggleMaximize() {
	switch row.state {
	case RowNormalState:
		row.maximize()
	case RowMinimizedState:
		row.maximize()
	case RowMaximizedState:
		row.unmaximize()
	}
}
func (row *Row) maximize() {
	row.setRowsState(RowMinimizedState)
	row.setState(RowMaximizedState)
	row.Col.CalcChildsBounds()
	row.Col.MarkNeedsPaint()
}
func (row *Row) unmaximize() {
	row.setRowsState(RowNormalState)
	row.Col.CalcChildsBounds()
	row.Col.MarkNeedsPaint()
}
func (row *Row) setRowsState(state RowState) {
	for _, r := range row.Col.Rows() {
		r.setState(state)
	}
}
func (row *Row) setState(state RowState) {
	row.state = state
	hide := state == RowMinimizedState
	row.scrollArea.SetHidden(hide)
}

type RowState int

const (
	RowNormalState RowState = iota
	RowMaximizedState
	RowMinimizedState
)

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
