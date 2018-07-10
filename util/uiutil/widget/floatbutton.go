package widget

import (
	"image"

	"github.com/jmigpin/editor/util/uiutil/event"
)

type FloatButton struct {
	*Button
	floatBox *FloatBox // added to the menulayer
}

func NewFloatButton(ctx ImageContext, ml *MultiLayer, content Node) *FloatButton {
	b := NewButton(ctx)
	b.Sticky = true
	b.Label.Text.SetStr("floatbutton")

	fb := &FloatButton{Button: b}

	// floatbox
	fb.floatBox = NewFloatBox(ml, content)
	fb.floatBox.Marks.Add(MarkForceZeroBounds)

	fb.OnClick = func(*event.MouseClick) {
		// toggle
		show := fb.floatBox.Marks.HasAny(MarkForceZeroBounds)
		if show {
			fb.floatBox.Marks.Remove(MarkForceZeroBounds)
			fb.floatBox.MarkNeedsLayoutAndPaint()
		} else {
			// hide
			fb.floatBox.Marks.Add(MarkForceZeroBounds)
			fb.floatBox.MarkNeedsLayout()
			ml.BgLayer.RectNeedsPaint(fb.floatBox.Bounds)
		}
	}

	return fb
}

func (fb *FloatButton) Close() {
	// remove floatbox from the menu layer
	fb.floatBox.Parent.Remove(fb.floatBox)
}

//----------

func (fb *FloatButton) Layout() {
	fb.Button.Layout()

	// update refpoint
	fb.floatBox.RefPoint = image.Point{fb.Bounds.Min.X, fb.Bounds.Max.Y}

	if !fb.floatBox.Marks.HasAny(MarkForceZeroBounds) {
		//fb.floatBox.Layout()
		fb.floatBox.MarkNeedsLayout()
	}
}

//----------
