package widget

import (
	"image"
	"image/color"

	"github.com/jmigpin/editor/util/imageutil"
)

type Rectangle struct {
	EmbedNode
	Size image.Point
	ctx  ImageContext
}

func NewRectangle(ctx ImageContext) *Rectangle {
	r := &Rectangle{ctx: ctx}
	return r
}
func (r *Rectangle) Measure(hint image.Point) image.Point {
	return r.Size
}
func (r *Rectangle) Paint() {
	r.paint(r.Theme.Palette().Normal.Bg)
}
func (r *Rectangle) paint(c color.Color) {
	imageutil.FillRectangle(r.ctx.Image(), &r.Bounds, c)
}
