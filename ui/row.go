package ui

import (
	"image"

	"github.com/jmigpin/editor/util/evreg"
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type Row struct {
	*widget.BoxLayout
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
	row.BoxLayout = widget.NewBoxLayout()
	row.YAxis = true

	row.EvReg = evreg.NewRegister()

	// row separator from other rows
	{
		row.sep = widget.NewRectangle(row.ui)
		row.sep.Size.Y = SeparatorWidth
		row.sep.Color = &SeparatorColor
		//row.sep.SetExpand(true, false)

		row.sepHandle.Init(row.ui, row.sep, row)
		row.sepHandle.Top = 3
		row.sepHandle.Bottom = 3
		row.sepHandle.Cursor = widget.MoveCursor
		row.Col.Cols.Layout.InsertRowSepHandle(&row.sepHandle)

		row.Append(row.sep)
		row.SetChildFill(row.sep, true, false)
	}

	// toolbar
	row.Toolbar = NewRowToolbar(row, NewToolbar(row.ui, row))
	row.Append(row.Toolbar)
	row.SetChildFlex(row.Toolbar, true, false)

	// scrollarea with textarea
	{
		row.TextArea = NewTextArea(row.ui)
		row.TextArea.Colors = &TextAreaColors
		row.scrollArea = widget.NewScrollArea(row.ui, row.TextArea, true, false)
		row.scrollArea.LeftScroll = ScrollbarLeft
		row.scrollArea.ScrollWidth = ScrollbarWidth
		row.scrollArea.VSBar.Color = &ScrollbarBgColor
		row.scrollArea.VSBar.Handle.Color = &ScrollbarFgColor
		if row.scrollArea.HSBar != nil {
			row.scrollArea.HSBar.Color = &ScrollbarBgColor
			row.scrollArea.HSBar.Handle.Color = &ScrollbarFgColor
		}

		var sa widget.Node = row.scrollArea
		if ShadowsOn {
			// scrollarea innershadow bellow the toolbar
			shadow := widget.NewShadow(row.ui, row.scrollArea)
			shadow.Top = ShadowSteps
			shadow.MaxShade = ShadowMaxShade
			sa = shadow
		} else {
			// toolbar/scrollarea separator
			tbSep := widget.NewRectangle(row.ui)
			tbSep.Size.Y = SeparatorWidth
			tbSep.Color = &RowInnerSeparatorColor
			row.Append(tbSep)
			row.SetChildFill(tbSep, true, false)
		}

		row.Append(sa)
		row.SetChildFlex(sa, true, true)
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
	// run callbacks first to allow the read the state before removing
	row.EvReg.RunCallbacks(RowCloseEventId, &RowCloseEvent{row})

	row.Col.Cols.Layout.Remove(&row.sepHandle)
	row.Col.removeRow(row)
}

func (row *Row) CalcChildsBounds() {
	row.BoxLayout.CalcChildsBounds()
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

func (row *Row) NextRow() *Row {
	u := row.Next()
	if u == nil {
		return nil
	}
	return u.(*Row)
}

func (row *Row) resizeWithMoveToPoint(p *image.Point) {
	col, ok := row.Col.Cols.PointColumnExtra(p)
	if !ok {
		return
	}

	// move to another column
	if col != row.Col {
		next, ok := col.PointNextRowExtra(p)
		if !ok {
			next = nil
		}
		row.Col.removeRow(row)
		col.insertRowBefore(row, next)
	}

	bounds := row.Col.Bounds
	dy := float64(bounds.Dy())
	perc := float64(p.Sub(bounds.Min).Y) / dy

	row.Col.RowsLayout.ResizeWithMove(row, perc)

	row.Col.CalcChildsBounds()
	row.Col.MarkNeedsPaint()
}

func (row *Row) Maximize() {
	col := row.Col
	col.RowsLayout.MaximizeNode(row)
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

	col.RowsLayout.ResizeWithPush(row, perc)

	col.CalcChildsBounds()
	col.MarkNeedsPaint()
}

func (row *Row) ResizeTextAreaIfVerySmall() {
	col := row.Col
	dy := float64(col.Bounds.Dy())
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
	col.RowsLayout.ResizeWithPush(row, perc)

	col.CalcChildsBounds()
	col.MarkNeedsPaint()

	// check if good already
	taDy = ta.Bounds.Dy()
	if taDy > taMin {
		return
	}

	// push siblings up
	perc = float64(row.Bounds.Max.Sub(col.Bounds.Min).Y-size) / dy
	col.RowsLayout.ResizeWithPush(row, perc)

	col.CalcChildsBounds()
	col.MarkNeedsPaint()
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
