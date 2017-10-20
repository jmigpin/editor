package widget

import (
	"image"
	"image/color"
)

// User Interface interfacer
type UIer interface {
	FillRectangle(*image.Rectangle, color.Color)
}

type UIStrDrawer interface {
	UIer
	MeasureString(string, image.Point) image.Point
	DrawString(string, *image.Rectangle, color.Color)
}
