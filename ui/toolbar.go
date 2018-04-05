package ui

import (
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type Toolbar struct {
	*TextArea
}

func NewToolbar(ui *UI, flexibleParent widget.Node) *Toolbar {
	tb := &Toolbar{}
	tb.TextArea = NewTextArea(ui)
	tb.TextArea.FlexibleParent = flexibleParent
	tb.TextArea.SetTheme(&UITheme.Toolbar)
	return tb
}
