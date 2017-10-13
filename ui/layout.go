package ui

import (
	"image"

	"github.com/jmigpin/editor/uiutil/widget"
)

type Layout struct {
	widget.FlowLayout
	UI      *UI
	Toolbar *Toolbar
	Cols    *Columns

	inEvNode widget.Node
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

func (l *Layout) SetInputEventNode(node widget.Node, v bool) {
	if v {
		l.inEvNode = node
	} else if l.inEvNode == node {
		l.inEvNode = nil
	}
}
func (l *Layout) OnInputEvent(ev interface{}, p image.Point) bool {
	// redirect input events to input node as requested
	if l.inEvNode != nil && !l.inEvNode.Hidden() {
		return l.inEvNode.OnInputEvent(ev, p)
	}

	return false
}
