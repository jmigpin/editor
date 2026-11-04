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
	fbb.floatBox = NewFloatBox(ml, content)
	fl.Append(fbb.floatBox)
	fbb.floatBox.Hide()

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

	refPoint := image.Point{fbb.Bounds.Min.X, fbb.Bounds.Max.Y}
	if fbb.floatBox.Visible() && fbb.floatBox.RefPoint != refPoint {
		fbb.floatBox.ml.AddMarkRect(fbb.floatBox.Bounds)
		fbb.floatBox.MarkNeedsLayout()
	}
	fbb.floatBox.RefPoint = refPoint
}
