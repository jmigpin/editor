package widget

import (
	"image"
	"image/color"
)

// User Interface interfacer
type UIer interface {
	FillRectangle(*image.Rectangle, color.Color)
}
