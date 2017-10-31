package ui

import (
	"image"

	"github.com/BurntSushi/xgbutil/xcursor"
	"github.com/jmigpin/editor/uiutil/event"
	"github.com/jmigpin/editor/uiutil/widget"
	"github.com/jmigpin/editor/xgbutil/evreg"
)

type Row struct {
	*widget.FlowLayout
	Square   *Square
	Toolbar  *Toolbar
	TextArea *TextArea
	Col      *Column
	EvReg    *evreg.Register

	scrollArea    *ScrollArea
	sep           widget.Rectangle
	closingCursor bool
}

func NewRow(col *Column) *Row {
	row := &Row{Col: col}
	row.FlowLayout = widget.NewFlowLayout()
	row.SetWrapper(row)

	ui := row.Col.Cols.Layout.UI

	row.EvReg = evreg.NewRegister()

	row.Toolbar = NewToolbar(ui, row)
	row.Toolbar.SetExpand(true, false)

	row.Square = NewSquare(ui)
	row.Square.SetFill(false, true)
	row.Square.EvReg.Add(SquareInputEventId, row.onSquareInput)

	// row separator from other rows
	row.sep.Init(ui)
	row.sep.SetExpand(true, false)
	row.sep.Size.Y = SeparatorWidth
	row.sep.Color = &SeparatorColor

	// square and toolbar
	tb := widget.NewFlowLayout()
	var sep1 widget.Rectangle
	sep1.Init(ui)
	sep1.Color = &RowInnerSeparatorColor
	sep1.Size.X = SeparatorWidth
	sep1.SetFill(false, true)
	if ScrollbarLeft {
		tb.Append(row.Square, &sep1, row.Toolbar)
	} else {
		tb.Append(row.Toolbar, &sep1, row.Square)
	}

	// toolbar separator from scrollarea
	var tbSep widget.Rectangle
	tbSep.Init(ui)
	tbSep.SetExpand(true, false)
	tbSep.Size.Y = SeparatorWidth
	tbSep.Color = &RowInnerSeparatorColor

	// scrollarea with textarea
	row.TextArea = NewTextArea(ui)
	row.TextArea.Colors = &TextAreaColors
	row.scrollArea = NewScrollArea(ui, row.TextArea)
	row.scrollArea.SetExpand(true, true)
	row.scrollArea.LeftScroll = ScrollbarLeft
	row.scrollArea.ScrollWidth = ScrollbarWidth
	row.scrollArea.VBar.Color = &ScrollbarBgColor
	row.scrollArea.VBar.Handle.Color = &ScrollbarFgColor

	row.YAxis = true
	row.Append(&row.sep, tb, &tbSep, row.scrollArea)

	return row
}
func (row *Row) activate() {
	if row.Square.Value(SquareActive) {
		return
	}
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
	row.EvReg.RunCallbacks(RowCloseEventId, &RowCloseEvent{row})
}

func (row *Row) onSquareInput(ev0 interface{}) {
	sqEv := ev0.(*SquareInputEvent)
	ui := row.Col.Cols.Layout.UI
	switch ev := sqEv.Event.(type) {
	case *event.MouseDown:
		switch ev.Button {
		case event.ButtonMiddle:
			row.closingCursor = true
			ui.CursorMan.SetCursor(xcursor.XCursor)
		case event.ButtonWheelUp:
			row.resizeWithPush(true)
		case event.ButtonWheelDown:
			row.resizeWithPush(false)
		}

	case *event.MouseClick:
		switch ev.Button {
		case event.ButtonLeft:
			row.maximize(&ev.Point)
		case event.ButtonMiddle:
			row.Close()
			ui.CursorMan.UnsetCursor()

		case event.ButtonWheelLeft:
			p2 := sqEv.TopPoint
			p2.X -= 20
			row.resizeColumnToPoint(p2)
			row.WarpPointer()
		case event.ButtonWheelRight:
			p2 := sqEv.TopPoint
			p2.X += 20
			row.resizeColumnToPoint(p2)
			row.WarpPointer()
		}

	case *event.MouseDragStart:
		if row.closingCursor {
			row.closingCursor = false
			ui.CursorMan.UnsetCursor()
		}
		switch ev.Button {
		case event.ButtonLeft:
			ui.CursorMan.SetCursor(xcursor.Fleur)
			row.resizeRowToPoint(sqEv.TopXPoint)
		case event.ButtonRight:
			ui.CursorMan.SetCursor(xcursor.SBHDoubleArrow)
			row.resizeColumnToPoint(sqEv.TopPoint)
		}
	case *event.MouseDragMove:
		switch {
		case ev.Buttons.Has(event.ButtonLeft):
			row.resizeRowToPoint(sqEv.TopXPoint)
		case ev.Buttons.Has(event.ButtonRight):
			row.resizeColumnToPoint(sqEv.TopPoint)
		}
	case *event.MouseDragEnd:
		switch ev.Button {
		case event.ButtonLeft:
			row.resizeRowToPoint(sqEv.TopXPoint)
			ui.CursorMan.UnsetCursor()
		case event.ButtonRight:
			row.resizeColumnToPoint(sqEv.TopPoint)
			ui.CursorMan.UnsetCursor()
		}
	}
}

func (row *Row) OnInputEvent(ev0 interface{}, p image.Point) bool {
	switch ev0.(type) {
	case *event.KeyDown:
		row.activate()
	case *event.MouseDown:
		row.activate()
	}

	ev2 := &RowInputEvent{row, ev0}
	row.EvReg.RunCallbacks(RowInputEventId, ev2)

	return false
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

func (row *Row) resizeRowToPoint(p *image.Point) {
	col, ok := row.Col.Cols.PointColumn(p)
	if !ok {
		return
	}
	if col != row.Col {
		// move to another column
		next, ok := col.PointRow(p)
		if ok {
			next, _ = next.NextRow()
		}
		if next != row {
			row.Col.removeRow(row)
			col.insertBefore(row, next)
		}
	}

	bounds := row.Col.Bounds()
	dy := float64(bounds.Dy())
	perc := float64(p.Sub(bounds.Min).Y) / dy
	min := float64(row.minimumSize()) / dy

	percIsTop := true
	rl := row.Col.RowsLayout
	rl.ResizeEndPercentWithSwap(row, perc, percIsTop, min)

	row.Col.CalcChildsBounds()
	row.Col.MarkNeedsPaint()
}
func (row *Row) resizeColumnToPoint(p *image.Point) {
	row.Col.resizeToPoint(p)
}

func (row *Row) maximize(p *image.Point) {
	col := row.Col
	dy := float64(col.Bounds().Dy())
	min := float64(row.minimumSize()) / dy
	col.RowsLayout.MaximizeEndPercentNode(row, min)
	col.CalcChildsBounds()
	col.MarkNeedsPaint()
	if !p.In(row.Square.Bounds()) {
		row.WarpPointer()
	}
}

func (row *Row) resizeWithPush(up bool) {
	col := row.Col
	dy := float64(col.Bounds().Dy())
	min := float64(row.minimumSize()) / dy

	jump := 30
	if up {
		jump *= -1
	}
	perc := float64(row.Bounds().Min.Y-col.Bounds().Min.Y+jump) / dy

	percIsTop := true
	col.RowsLayout.ResizeEndPercentWithPush(row, perc, percIsTop, min)

	col.CalcChildsBounds()
	col.MarkNeedsPaint()

	// keep pointer inside the square (newly calculated)
	row.WarpPointer()
}

func (row *Row) ResizeTextAreaIfVerySmall() {
	col := row.Col
	dy := float64(col.Bounds().Dy())
	min := float64(row.minimumSize()) / dy
	ta := row.TextArea
	taMin := ta.LineHeight()

	taDy := ta.Bounds().Dy()
	if taDy > taMin {
		return
	}

	hint := image.Point{row.Bounds().Dx(), 1000000} // TODO: use column dy?
	tbm := row.Toolbar.Measure(hint)
	size := tbm.Y + taMin + 10 // pad to cover borders used // TODO: improve by getting toolbar+border size from a func?

	// push siblings down
	perc := float64(row.Bounds().Min.Sub(col.Bounds().Min).Y+size) / dy
	percIsTop := false
	col.RowsLayout.ResizeEndPercentWithPush(row, perc, percIsTop, min)

	col.CalcChildsBounds()
	col.MarkNeedsPaint()

	// check if good already
	taDy = ta.Bounds().Dy()
	if taDy > taMin {
		return
	}

	// push siblings up
	perc = float64(row.Bounds().Max.Sub(col.Bounds().Min).Y-size) / dy
	percIsTop = true
	col.RowsLayout.ResizeEndPercentWithPush(row, perc, percIsTop, min)

	col.CalcChildsBounds()
	col.MarkNeedsPaint()
}

func (row *Row) minimumSize() int {
	return row.TextArea.LineHeight()
}

const (
	RowInputEventId = iota
	RowCloseEventId
)

type RowInputEvent struct {
	Row   *Row
	Event interface{}
}
type RowCloseEvent struct {
	Row *Row
}
