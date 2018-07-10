package widget

import (
	"image"
)

type TextScroll struct {
	*Text
}

//----------

// Implement widget.Scrollable

func (ts *TextScroll) ScrollableOffset() image.Point {
	return ts.Offset()
}

func (ts *TextScroll) SetScrollableOffset(o image.Point) {
	ts.SetOffset(o)
}

func (ts *TextScroll) ScrollableSize() image.Point {
	// extra height allows to scroll past the str height
	visible := 2 * ts.LineHeight() // keep n lines visible at the end
	extra := ts.Embed().Bounds.Dy() - visible

	m := ts.FullMeasurement()
	m.Y += extra
	return m
}

func (ts *TextScroll) ScrollablePagingMargin() int {
	return ts.LineHeight() * 1
}

func (ts *TextScroll) ScrollableScrollJump() int {
	return ts.LineHeight() * 4
}
