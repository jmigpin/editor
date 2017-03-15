package ui

import "github.com/jmigpin/editor/xutil/xgbutil"

type Toolbar struct {
	*TextArea
	OnSetText func()
}

func NewToolbar(ui *UI) *Toolbar {
	tb := &Toolbar{TextArea: NewTextArea(ui)}
	tb.DisableHighlightCursorWord = true
	tb.DisableButtonScroll = true

	fn := &xgbutil.ERCallback{tb.onTextAreaSetText}
	tb.TextArea.EvReg.Add(TextAreaSetTextEventId, fn)

	return tb
}
func (tb *Toolbar) onTextAreaSetText(ev0 xgbutil.EREvent) {
	ev := ev0.(*TextAreaSetTextEvent)
	if tb.OnSetText != nil {
		tb.OnSetText()
	}
	// keep pointer inside if it was in before
	// useful in dynamic bounds becoming shorter and leaving the pointer outside, losing keyboard focus
	p, ok := tb.ui.Win.QueryPointer()
	if ok && p.In(ev.OldBounds) {
		tb.ui.WarpPointerToRectangle(&tb.C.Bounds)
	}
}
