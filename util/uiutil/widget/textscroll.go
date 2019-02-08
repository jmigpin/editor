package widget

import (
	"image"

	"github.com/jmigpin/editor/util/drawutil/drawer4"
)

type TextScroll struct {
	*Text
}

//----------

// Implement widget.Scrollable

func (ts *TextScroll) ScrollableOffset() image.Point {
	if d, ok := ts.Drawer.(*drawer4.Drawer); ok {
		return image.Point{1, d.RuneOffset()}
	}

	return ts.Offset()
}

func (ts *TextScroll) SetScrollableOffset(o image.Point) {
	if d, ok := ts.Drawer.(*drawer4.Drawer); ok {
		if d.Opt.RuneOffset.On {
			d.SetRuneOffset(o.Y)
			// TODO: check if it differs
			ts.MarkNeedsLayoutAndPaint()
			return
		}
	}

	ts.SetOffset(o)
}

func (ts *TextScroll) ScrollableSize() image.Point {
	if _, ok := ts.Drawer.(*drawer4.Drawer); ok {
		return ts.FullMeasurement()
	}

	//	// extra height allows to scroll past the str height
	//	visible := 2 * ts.LineHeight() // keep n lines visible at the end
	//	extra := ts.Embed().Bounds.Dy() - visible

	//	m := ts.FullMeasurement()
	//	m.Y += extra
	//	return m

	return ts.FullMeasurement()
}

func (ts *TextScroll) ScrollableViewSize() image.Point {
	if d, ok := ts.Drawer.(*drawer4.Drawer); ok {
		return d.ScrollableViewSize()
	}
	return ts.Bounds.Size()
}

func (ts *TextScroll) ScrollablePagingMargin() int {
	return ts.LineHeight() * 1
}

func (ts *TextScroll) ScrollableScrollJump() int {
	return ts.LineHeight() * 4
}
