package widget

import (
	"image"
	"image/color"
)

type Rectangle struct {
	EmbedNode
	Color color.Color
	Size  image.Point
	ui    UIer
}

func NewRectangle(ui UIer) *Rectangle {
	return &Rectangle{
		ui:   ui,
		Size: image.Point{10, 10}, // this size is used in tests
	}
}
func (r *Rectangle) Measure(max image.Point) image.Point {
	return r.Size
}
func (r *Rectangle) CalcChildsBounds() {
}
func (r *Rectangle) Paint() {
	if r.Color != nil {
		b := r.Bounds()
		r.ui.FillRectangle(&b, r.Color)
	}
}
