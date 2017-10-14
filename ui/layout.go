package ui

import "github.com/jmigpin/editor/uiutil/widget"

type Layout struct {
	widget.FlowLayout
	UI      *UI
	Toolbar *Toolbar
	Cols    *Columns
}

func NewLayout(ui *UI) *Layout {
	layout := &Layout{}
	layout.UI = ui

	layout.Toolbar = NewToolbar(ui, layout)
	layout.Toolbar.SetExpand(true, false)

	sep := widget.NewSpace(ui)
	sep.SetExpand(true, false)
	sep.Size.Y = SeparatorWidth
	sep.Color = SeparatorColor

	layout.Cols = NewColumns(layout)
	layout.Cols.SetExpand(true, true)

	layout.YAxis = true
	widget.AppendChilds(layout, layout.Toolbar, sep, layout.Cols)

	return layout
}
