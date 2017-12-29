package ui

import (
	"image"

	"github.com/jmigpin/editor/uiutil/event"
	"github.com/jmigpin/editor/uiutil/widget"
	"github.com/jmigpin/editor/xgbutil/evreg"
)

type Row struct {
	*widget.FlowLayout
	Toolbar  *RowToolbar
	TextArea *TextArea
	Col      *Column
	EvReg    *evreg.Register

	scrollArea *widget.ScrollArea
	sep        *widget.Rectangle
	sepHandle  RowSeparatorHandle
	ui         *UI
}

func NewRow(col *Column) *Row {
	row := &Row{Col: col, ui: col.Cols.Layout.UI}
	row.FlowLayout = widget.NewFlowLayout()
	row.SetWrapper(row)

	row.EvReg = evreg.NewRegister()

	row.Toolbar = NewRowToolbar(row, NewToolbar(row.ui, row))
	row.Toolbar.SetExpand(true, false)

	// row separator from other rows
	row.sep = widget.NewRectangle(row.ui)
	row.sep.SetExpand(true, false)
	row.sep.Size.Y = SeparatorWidth
	row.sep.Color = &SeparatorColor

	row.sepHandle.Init(row.ui, row.sep, row)
	row.sepHandle.Top = 3
	row.sepHandle.Bottom = 3
	row.sepHandle.Cursor = widget.MoveCursor
	row.Col.Cols.Layout.InsertRowSepHandle(&row.sepHandle)

	// scrollarea with textarea
	row.TextArea = NewTextArea(row.ui)
	row.TextArea.Colors = &TextAreaColors
	row.scrollArea = widget.NewScrollArea(row.ui, row.TextArea, true, false)
	row.scrollArea.SetExpand(true, true)
	row.scrollArea.LeftScroll = ScrollbarLeft
	row.scrollArea.ScrollWidth = ScrollbarWidth
	row.scrollArea.VSBar.Color = &ScrollbarBgColor
	row.scrollArea.VSBar.Handle.Color = &ScrollbarFgColor
	if row.scrollArea.HSBar != nil {
		row.scrollArea.HSBar.Color = &ScrollbarBgColor
		row.scrollArea.HSBar.Handle.Color = &ScrollbarFgColor
	}

	row.YAxis = true
	row.Append(row.sep, row.Toolbar)

	if ShadowsOn {
		// scrollarea innershadow bellow the toolbar
		shadow := widget.NewShadow(row.ui, row.scrollArea)
		shadow.Top = ShadowSteps
		shadow.MaxShade = ShadowMaxShade

		row.Append(shadow)
	} else {
		// toolbar/scrollarea separator
		tbSep := widget.NewRectangle(row.ui)
		tbSep.SetExpand(true, false)
		tbSep.Size.Y = SeparatorWidth
		tbSep.Color = &RowInnerSeparatorColor

		row.Append(tbSep, row.scrollArea)
	}

	return row
}

func (row *Row) activate() {
	if row.HasState(ActiveRowState) {
		return
	}
	// deactivate previous active row
	for _, c := range row.Col.Cols.Columns() {
		for _, r := range c.Rows() {
			r.SetState(ActiveRowState, false)
		}
	}
	// activate this row
	row.SetState(ActiveRowState, true)
}

func (row *Row) Close() {
	row.Col.Cols.Layout.Remove(&row.sepHandle)
	row.Col.removeRow(row)
	row.EvReg.RunCallbacks(RowCloseEventId, &RowCloseEvent{row})
}

func (row *Row) CalcChildsBounds() {
	row.FlowLayout.CalcChildsBounds()
	row.sepHandle.CalcChildsBounds()
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

	bounds := row.Col.Bounds
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
	dy := float64(col.Bounds.Dy())
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

	pad := p.Sub(row.Bounds.Min)

	p2 := row.Bounds.Min
	p2.Y += jump
	row.resizeWithPushToPoint(&p2)

	// keep same pad since using it as well when moving from the square
	p3 := row.Bounds.Min.Add(pad)
	row.ui.WarpPointer(&p3)
}
func (row *Row) resizeWithPushToPoint(p *image.Point) {
	col := row.Col
	dy := float64(col.Bounds.Dy())
	perc := float64(p.Sub(col.Bounds.Min).Y) / dy
	min := float64(row.minimumSize()) / dy

	percIsTop := true
	col.RowsLayout.ResizeEndPercentWithPush(row, perc, percIsTop, min)

	col.CalcChildsBounds()
	col.MarkNeedsPaint()
}

func (row *Row) ResizeTextAreaIfVerySmall() {
	col := row.Col
	dy := float64(col.Bounds.Dy())
	min := float64(row.minimumSize()) / dy
	ta := row.TextArea
	taMin := ta.LineHeight()

	taDy := ta.Bounds.Dy()
	if taDy > taMin {
		return
	}

	hint := image.Point{row.Bounds.Dx(), 1000000} // TODO: use column dy?
	tbm := row.Toolbar.Measure(hint)
	size := tbm.Y + taMin + 10 // pad to cover borders used // TODO: improve by getting toolbar+border size from a func?

	// push siblings down
	perc := float64(row.Bounds.Min.Sub(col.Bounds.Min).Y+size) / dy
	percIsTop := false
	col.RowsLayout.ResizeEndPercentWithPush(row, perc, percIsTop, min)

	col.CalcChildsBounds()
	col.MarkNeedsPaint()

	// check if good already
	taDy = ta.Bounds.Dy()
	if taDy > taMin {
		return
	}

	// push siblings up
	perc = float64(row.Bounds.Max.Sub(col.Bounds.Min).Y-size) / dy
	percIsTop = true
	col.RowsLayout.ResizeEndPercentWithPush(row, perc, percIsTop, min)

	col.CalcChildsBounds()
	col.MarkNeedsPaint()
}

func (row *Row) minimumSize() int {
	return row.TextArea.LineHeight()
}

func (row *Row) SetState(s RowState, v bool) {
	row.Toolbar.Square.SetState(s, v)
}
func (row *Row) HasState(s RowState) bool {
	return row.Toolbar.Square.HasState(s)
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
