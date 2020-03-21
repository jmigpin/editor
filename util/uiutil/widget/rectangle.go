package widget

import (
	"image"

	"github.com/jmigpin/editor/util/imageutil"
)

type Rectangle struct {
	ENode
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
	bg := r.TreeThemePaletteColor("rect")
	imageutil.FillRectangle(r.ctx.Image(), r.Bounds, bg)
}
