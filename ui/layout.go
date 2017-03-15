package ui

import "github.com/jmigpin/editor/uiutil"

type Layout struct {
	C       uiutil.Container
	UI      *UI
	Toolbar *Toolbar
	Cols    *Columns
	Sep     uiutil.Container
}

func NewLayout(ui *UI) *Layout {
	layout := &Layout{}
	layout.UI = ui

	layout.Toolbar = NewToolbar(ui)
	tb := layout.Toolbar
	tb.Colors = &ToolbarColors

	sep := NewSeparator(ui, SeparatorWidth, SeparatorColor)

	layout.Cols = NewColumns(layout)

	layout.C.Style.Direction = uiutil.ColumnDirection
	layout.C.AppendChilds(&layout.Toolbar.C, &sep.C, &layout.Cols.C)

	// dynamic toolbar bounds
	tb.C.Style.DynamicMainSize = func() int {
		return tb.CalcStringHeight(layout.C.Bounds.Dx())
	}
	tb.OnSetText = func() {
		b := tb.C.Bounds
		layout.C.CalcChildsBounds()
		if !tb.C.Bounds.Eq(b) {
			layout.C.NeedPaint()
		}
	}

	return layout
}
