package ui

import (
	"image"

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

	scrollArea *ScrollArea
	sep        widget.Rectangle
	sepHandle  RowSeparatorHandle
	ui         *UI
}

func NewRow(col *Column) *Row {
	row := &Row{Col: col, ui: col.Cols.Layout.UI}
	row.FlowLayout = widget.NewFlowLayout()
	row.SetWrapper(row)

	row.EvReg = evreg.NewRegister()

	row.Toolbar = NewToolbar(row.ui, row)
	row.Toolbar.SetExpand(true, false)

	row.Square = NewSquare(row.ui)
	row.Square.SetFill(false, true)
	row.Square.EvReg.Add(SquareInputEventId, row.onSquareInput)

	// row separator from other rows
	row.sep.Init(row.ui)
	row.sep.SetExpand(true, false)
	row.sep.Size.Y = SeparatorWidth
	row.sep.Color = &SeparatorColor

	row.sepHandle.Init(row.ui, &row.sep, row)
	row.sepHandle.Top = 3
	row.sepHandle.Bottom = 3
	row.sepHandle.Cursor = widget.MoveCursor
	row.Col.Cols.Layout.InsertRowSepHandle(&row.sepHandle)

	// square and toolbar
	tb := widget.NewFlowLayout()
	var sep1 widget.Rectangle
	sep1.Init(row.ui)
	sep1.Color = &RowInnerSeparatorColor
	sep1.Size.X = SeparatorWidth
	sep1.SetFill(false, true)
	tb.Append(row.Square, &sep1, row.Toolbar)

	// scrollarea with textarea
	row.TextArea = NewTextArea(row.ui)
	row.TextArea.Colors = &TextAreaColors
	row.scrollArea = NewScrollArea(row.ui, row.TextArea)
	row.scrollArea.SetExpand(true, true)
	row.scrollArea.LeftScroll = ScrollbarLeft
	row.scrollArea.ScrollWidth = ScrollbarWidth
	row.scrollArea.VBar.Color = &ScrollbarBgColor
	row.scrollArea.VBar.Handle.Color = &ScrollbarFgColor

	row.YAxis = true
	row.Append(&row.sep, tb)

	if ShadowsOn {
		// scrollarea innershadow bellow the toolbar
		var shadow widget.Shadow
		shadow.Init(row.ui, row.scrollArea)
		shadow.Top = ShadowSteps
		shadow.MaxShade = ShadowMaxShade

		row.Append(&shadow)
	} else {
		// toolbar/scrollarea separator
		var tbSep widget.Rectangle
		tbSep.Init(row.ui)
		tbSep.SetExpand(true, false)
		tbSep.Size.Y = SeparatorWidth
		tbSep.Color = &RowInnerSeparatorColor

		row.Append(&tbSep, row.scrollArea)
	}

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
	row.TextArea.UnsetPointerCursor(row.ui)
	row.Col.Cols.Layout.Remove(&row.sepHandle)
	row.Col.removeRow(row)
	row.EvReg.RunCallbacks(RowCloseEventId, &RowCloseEvent{row})
}

func (row *Row) CalcChildsBounds() {
	row.FlowLayout.CalcChildsBounds()
	row.sepHandle.CalcChildsBounds()
}

func (row *Row) onSquareInput(ev0 interface{}) {
	sqEv := ev0.(*SquareInputEvent)
	switch ev := sqEv.Event.(type) {
	case *event.MouseEnter:
		row.SetPointerCursor(row.ui, widget.CloseCursor)
	case *event.MouseLeave:
		row.UnsetPointerCursor(row.ui)
	case *event.MouseClick:
		switch ev.Button {
		case event.ButtonLeft:
			row.UnsetPointerCursor(row.ui)
			row.Close()
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

// Safe to use concurrently.
func (row *Row) Flash() {
	row.Toolbar.Flash() // Safe to use concurrently.
}

func (row *Row) NextRow() (*Row, bool) {
	u := row.Next()
	if u == nil {
		return nil, false
	}
	return u.(*Row), true
}

func (row *Row) resizeWithSwapToPoint(p *image.Point) {
	col, ok := row.Col.Cols.PointColumnExtra(p)
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

func (row *Row) Maximize() {
	col := row.Col
	dy := float64(col.Bounds().Dy())
	min := float64(row.minimumSize()) / dy
	col.RowsLayout.MaximizeEndPercentNode(row, min)
	col.CalcChildsBounds()
	col.MarkNeedsPaint()
}

func (row *Row) resizeWithPushJump(up bool, p *image.Point) {
	jump := 30
	if up {
		jump *= -1
	}

	pad := p.Sub(row.Bounds().Min)

	p2 := row.Bounds().Min
	p2.Y += jump
	row.resizeWithPushToPoint(&p2)

	// keep same pad since using it as well when moving from the square
	p3 := row.Bounds().Min.Add(pad)
	row.ui.WarpPointer(&p3)
}
func (row *Row) resizeWithPushToPoint(p *image.Point) {
	col := row.Col
	dy := float64(col.Bounds().Dy())
	perc := float64(p.Sub(col.Bounds().Min).Y) / dy
	min := float64(row.minimumSize()) / dy

	percIsTop := true
	col.RowsLayout.ResizeEndPercentWithPush(row, perc, percIsTop, min)

	col.CalcChildsBounds()
	col.MarkNeedsPaint()
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

type RowSeparatorHandle struct {
	widget.SeparatorHandle
	row *Row
}

func (sh *RowSeparatorHandle) Init(ctx widget.Context, ref widget.Node, row *Row) {
	sh.SeparatorHandle.Init(ctx, ref)
	sh.SetWrapper(sh)
	sh.row = row
}
func (sh *RowSeparatorHandle) OnInputEvent(ev0 interface{}, p image.Point) bool {
	_ = sh.SeparatorHandle.OnInputEvent(ev0, p)
	if sh.Dragging {
		sh.row.resizeWithSwapToPoint(&p)
	}
	switch ev := ev0.(type) {
	case *event.MouseDown:
		switch ev.Button {
		case event.ButtonWheelUp:
			sh.row.resizeWithPushJump(true, &p)
		case event.ButtonWheelDown:
			sh.row.resizeWithPushJump(false, &p)
		}
	}
	return false
}
