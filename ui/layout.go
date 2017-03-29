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

	layout.Toolbar = NewToolbar(ui, &layout.C)

	sep := NewSeparator(ui, SeparatorWidth, SeparatorColor)

	layout.Cols = NewColumns(layout)

	layout.C.Style.Direction = uiutil.ColumnDirection
	layout.C.AppendChilds(&layout.Toolbar.C, &sep.C, &layout.Cols.C)

	// dynamic toolbar bounds
	layout.Toolbar.C.Style.DynamicMainSize = func() int {
		return layout.Toolbar.CalcStringHeight(layout.C.Bounds.Dx())
	}

	return layout
}
