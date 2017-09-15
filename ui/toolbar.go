package ui

import (
	"github.com/jmigpin/editor/uiutil"
	"github.com/jmigpin/editor/xgbutil"
)

type Toolbar struct {
	*TextArea
	parentC *uiutil.Container
}

func NewToolbar(ui *UI, parentC *uiutil.Container) *Toolbar {
	tb := &Toolbar{TextArea: NewTextArea(ui), parentC: parentC}
	tb.DisableHighlightCursorWord = true
	tb.DisablePageUpDown = true

	tb.Colors = &ToolbarColors

	tb.TextArea.EvReg.Add(TextAreaSetStrEventId,
		&xgbutil.ERCallback{tb.onTextAreaSetStr})

	return tb
}
func (tb *Toolbar) onTextAreaSetStr(ev0 interface{}) {
	ev := ev0.(*TextAreaSetStrEvent)

	// dynamic toolbar bounds
	// if toolbar bounds changed due to text change (dynamic height) then the parent container needs paint
	b := tb.C.Bounds
	tb.parentC.CalcChildsBounds()
	if !tb.C.Bounds.Eq(b) {
		tb.parentC.NeedPaint()
	}

	// keep pointer inside if it was in before
	// useful in dynamic bounds becoming shorter and leaving the pointer outside, losing keyboard focus
	p, ok := tb.ui.QueryPointer()
	if ok && p.In(ev.OldBounds) && !p.In(tb.C.Bounds) {
		tb.ui.WarpPointerToRectanglePad(&tb.C.Bounds)
	}
}
