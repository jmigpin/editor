package widget

import (
	"image"

	"github.com/jmigpin/editor/util/uiutil/event"
)

type FloatBoxButton struct {
	*Button
	floatBox *FloatBox // added to the menulayer
}

func NewFloatBoxButton(ctx ImageContext, ml *MultiLayer, fl *FloatLayer, content Node) *FloatBoxButton {
	b := NewButton(ctx)
	b.Sticky = true
	b.Label.Text.SetStr("floatboxbutton")

	fbb := &FloatBoxButton{Button: b}

	// floatbox
	fbb.floatBox = NewFloatBox(ml, fl, content)
	fbb.floatBox.Marks.Add(MarkForceZeroBounds)

	fbb.OnClick = func(*event.MouseClick) {
		fbb.floatBox.Toggle()
	}

	return fbb
}

func (fbb *FloatBoxButton) Close() {
	// remove floatbox from the floatlayer
	fbb.floatBox.Parent.Remove(fbb.floatBox)
}

//----------

func (fbb *FloatBoxButton) Layout() {
	fbb.Button.Layout()

	// update refpoint
	fbb.floatBox.RefPoint = image.Point{fbb.Bounds.Min.X, fbb.Bounds.Max.Y}

	if !fbb.floatBox.Marks.HasAny(MarkForceZeroBounds) {
		//fbb.floatBox.Layout()
		fbb.floatBox.MarkNeedsLayout()
	}
}
