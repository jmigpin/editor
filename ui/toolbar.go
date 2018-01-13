package ui

import (
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type Toolbar struct {
	*TextArea
	upperNode widget.Node
}

func NewToolbar(ui *UI, upperNode widget.Node) *Toolbar {
	tb := &Toolbar{upperNode: upperNode}
	tb.TextArea = NewTextArea(ui)
	tb.TextArea.Theme = &DefaultUITheme.ToolbarTheme
	tb.TextArea.EvReg.Add(TextAreaSetStrEventId, tb.onTextAreaSetStr)
	return tb
}
func (tb *Toolbar) onTextAreaSetStr(ev0 interface{}) {
	//ev := ev0.(*TextAreaSetStrEvent)

	// dynamic toolbar bounds that change when edited
	// if toolbar bounds changed due to text change (dynamic height) then the upper node container needs paint
	b := tb.Bounds
	tb.upperNode.CalcChildsBounds()
	if !tb.Bounds.Eq(b) {
		tb.upperNode.Embed().MarkNeedsPaint()
	}

	// TODO: move this to the textarea? (check dynamic flag)

	// Keep pointer inside if it was in before. Need to test if it was in before, otherwise it will warp the pointer on any change.
	// Useful in dynamic bounds becoming shorter and leaving the pointer outside, losing keyboard focus.
	b2 := tb.Bounds // new recalc'ed bounds
	p, err := tb.ui.QueryPointer()
	if err == nil && p.In(b) && !p.In(b2) {
		tb.ui.WarpPointerToRectanglePad(&b2)
	}
}
