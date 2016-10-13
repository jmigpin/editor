package ui

import "image"

type Toolbar struct {
	*TextArea
	Data interface{} // external use
}

func NewToolbar() *Toolbar {
	tb := &Toolbar{TextArea: NewTextArea()}
	tb.TextArea.Data = tb
	tb.DisableHighlightCursorWord = true
	tb.DisableButtonScroll = true
	return tb
}
func (tb *Toolbar) CalcArea(area *image.Rectangle) {
	tb.TextArea.CalcArea(area)
	tb.Area.Max.Y = tb.TextArea.UsedY()
}
