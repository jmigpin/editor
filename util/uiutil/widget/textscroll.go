package widget

import (
	"image"
)

type TextScroll struct {
	*Text
}

//----------
// Implement widget.Scrollable

func (ts *TextScroll) ScrollOffset() image.Point {
	return ts.Drawer.ScrollOffset()
}

func (ts *TextScroll) SetScrollOffset(o image.Point) {
	if ts.Drawer.ScrollOffset() != o {
		ts.Drawer.SetScrollOffset(o)
		ts.MarkNeedsLayoutAndPaint()
	}
}

func (ts *TextScroll) ScrollSize() image.Point {
	return ts.Drawer.ScrollSize()
}

func (ts *TextScroll) ScrollViewSize() image.Point {
	return ts.Drawer.ScrollViewSize()
}

func (ts *TextScroll) ScrollPageSizeY(up bool) int {
	return ts.Drawer.ScrollPageSizeY(up)
}

func (ts *TextScroll) ScrollWheelSizeY(up bool) int {
	return ts.Drawer.ScrollWheelSizeY(up)
}
