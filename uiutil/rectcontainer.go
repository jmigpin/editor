package uiutil

import (
	"image/color"
	"image/draw"

	"github.com/jmigpin/editor/imageutil"
)

// Simple container to draw a colored rectangle.
type RectContainer struct {
	C     Container
	imgFn func() draw.Image
	Color color.Color
}

func NewRectContainer(imgFn func() draw.Image) *RectContainer {
	rc := &RectContainer{imgFn: imgFn, Color: color.White}
	rc.C.PaintFunc = rc.paint
	return rc
}
func (rc *RectContainer) paint() {
	img := rc.imgFn()
	imageutil.FillRectangle(img, &rc.C.Bounds, rc.Color)
}
