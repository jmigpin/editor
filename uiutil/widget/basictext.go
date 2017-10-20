package widget

import (
	"image"
	"image/color"
)

type BasicText struct {
	LeafEmbedNode
	Str   string
	Color color.Color
	ui    UIStrDrawer
}

func (bt *BasicText) Init(ui UIStrDrawer) {
	bt.ui = ui
	bt.Color = color.Black
}
func (bt *BasicText) Measure(hint image.Point) image.Point {
	return bt.ui.MeasureString(bt.Str, hint)
}
func (bt *BasicText) Paint() {
	u := bt.Bounds()
	bt.ui.DrawString(bt.Str, &u, bt.Color)
}
