package ui

import (
	"github.com/jmigpin/editor/uiutil/widget"
	"github.com/jmigpin/editor/xgbutil/evreg"
)

type Toolbar struct {
	TextArea
	parent widget.Node
}

func NewToolbar(ui *UI, parent widget.Node) *Toolbar {
	tb := &Toolbar{parent: parent}
	tb.TextArea.Init(ui)

	tb.DisableHighlightCursorWord = true
	tb.DisablePageUpDown = true

	tb.Colors = &ToolbarColors

	tb.TextArea.EvReg.Add(TextAreaSetStrEventId,
		&evreg.Callback{tb.onTextAreaSetStr})

	return tb
}
func (tb *Toolbar) onTextAreaSetStr(ev0 interface{}) {
	ev := ev0.(*TextAreaSetStrEvent)

	// dynamic toolbar bounds that change when edited
	// if toolbar bounds changed due to text change (dynamic height) then the parent container needs paint
	b := tb.Bounds()
	tb.parent.CalcChildsBounds()
	if !tb.Bounds().Eq(b) {
		tb.parent.MarkNeedsPaint()
	}

	// keep pointer inside if it was in before -- need to test if it was in before otherwise and auto-change that edits the toolbar will warp the pointer
	// useful in dynamic bounds becoming shorter and leaving the pointer outside, losing keyboard focus
	p, ok := tb.ui.QueryPointer()
	if ok && p.In(ev.OldBounds) && !p.In(b) {
		tb.ui.WarpPointerToRectanglePad(&b)
	}
}
