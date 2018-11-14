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

	ScrollArea *widget.ScrollArea
	sep        *RowSeparator
	ui         *UI
}

func NewRow(col *Column) *Row {
	row := &Row{Col: col, ui: col.Cols.Root.UI}
	row.BoxLayout = widget.NewBoxLayout()
	row.YAxis = true

	row.EvReg = evreg.NewRegister()

	// row separator from other rows
	row.sep = NewRowSeparator(row)
	row.Append(row.sep)
	row.SetChildFill(row.sep, true, false)

	// toolbar
	row.Toolbar = NewRowToolbar(row)
	row.Append(row.Toolbar)
	row.SetChildFlex(row.Toolbar, true, false)

	// scrollarea with textarea
	{
		row.TextArea = NewTextArea(row.ui)
		row.TextArea.EnableWrapLines(true)
		row.TextArea.EnableHighlightCursorWord(true)
		row.TextArea.EnableColorizeSyntax(true)
		row.TextArea.EnableParenthesisMatch(true)

		row.ScrollArea = widget.NewScrollArea(row.ui, row.TextArea, false, true)
		row.ScrollArea.LeftScroll = ScrollBarLeft

		container := WrapInTopShadowOrSeparator(row.ui, row.ScrollArea)
		row.Append(container)
		row.SetChildFlex(container, true, true)
	}

	return row
}

//----------

func (row *Row) Close() {
	row.Col.removeRow(row)
	row.Col = nil
	row.sep.Close()
	row.EvReg.RunCallbacks(RowCloseEventId, &RowCloseEvent{row})
}

//----------

func (row *Row) activate() {
	if row.HasState(RowStateActive) {
		return
	}
	// deactivate previous active row
	for _, c := range row.Col.Cols.Columns() {
		for _, r := range c.Rows() {
			if r != row {
				r.SetState(RowStateActive, false)
			}
		}
	}
	// activate this row
	row.SetState(RowStateActive, true)
}

//----------

func (row *Row) OnChildMarked(child widget.Node, newMarks widget.Marks) {
	// dynamic toolbar
	if row.Toolbar != nil && row.Toolbar.HasAnyMarks(widget.MarkNeedsLayout) {
		row.MarkNeedsLayout()
	}
}

//----------

func (row *Row) Layout() {
	row.ScrollArea.ScrollWidth = UIThemeUtil.GetScrollBarWidth(row.TextArea.TreeThemeFont())
	row.BoxLayout.Layout()
}

//----------

func (row *Row) OnInputEvent(ev0 interface{}, p image.Point) event.Handle {
	switch ev0.(type) {
	case *event.KeyDown:
		row.activate()
	case *event.MouseDown:
		row.activate()
	}

	ev2 := &RowInputEvent{row, ev0}
	row.EvReg.RunCallbacks(RowInputEventId, ev2)

	return event.NotHandled
}

//----------

func (row *Row) NextRow() *Row {
	u := row.NextSiblingWrapper()
	if u == nil {
		return nil
	}
	return u.(*Row)
}

//----------

func (row *Row) Maximize() {
	col := row.Col
	col.RowsLayout.Spl.MaximizeNode(row)
}

//----------

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

	row.Col.RowsLayout.Spl.ResizeWithMove(row, perc)
}

//----------

func (row *Row) resizeWithPushJump(up bool, p *image.Point) {
	jump := 40
	if up {
		jump *= -1
	}

	pad := p.Sub(row.Bounds.Min)

	p2 := row.Bounds.Min
	p2.Y += jump
	row.resizeWithPushToPoint(&p2)

	// layout for accurate bounds, to warp pointer
	row.Col.RowsLayout.Spl.Layout()

	p3 := row.Bounds.Min.Add(pad)
	p3.Y = row.Bounds.Min.Y // accurate y
	row.ui.WarpPointer(&p3)
}

func (row *Row) resizeWithPushToPoint(p *image.Point) {
	col := row.Col
	dy := float64(col.Bounds.Dy())
	perc := float64(p.Sub(col.Bounds.Min).Y) / dy

	col.RowsLayout.Spl.SetPercentWithPush(row, perc)
}

//----------

func (row *Row) EnsureTextAreaMinimumHeight() {
	ta := row.TextArea

	taMin := ta.LineHeight() * 3
	if ta.Bounds.Dy() >= taMin {
		return
	}

	hint := image.Point{row.Bounds.Dx(), row.Col.Bounds.Dy()}
	tbm := row.Toolbar.Measure(hint)
	minH := tbm.Y + taMin + 2 // pad to cover borders used
	perc := float64(minH) / float64(row.Col.Bounds.Dy())

	row.Col.RowsLayout.Spl.SetSizePercentWithPush(row, perc)
}

//----------

func (row *Row) SetState(s RowState, v bool) {
	row.Toolbar.Square.SetState(s, v)
}
func (row *Row) HasState(s RowState) bool {
	return row.Toolbar.Square.HasState(s)
}

//----------

func (row *Row) PosBelow() *RowPos {
	return NewRowPos(row.Col, row.NextRow())
}

//----------

func (row *Row) ToggleTextAreaXBar() {
	enabled := row.IsTextAreaXBarEnabled()
	row.EnableTextAreaXBar(!enabled)
}

func (row *Row) IsTextAreaXBarEnabled() bool {
	return row.ScrollArea.XBar != nil
}

func (row *Row) EnableTextAreaXBar(v bool) {
	row.ScrollArea.SetBars(v, true)
}

//----------

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
