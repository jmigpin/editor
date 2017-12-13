package widget

import (
	"image"
	"image/color"

	"github.com/jmigpin/editor/imageutil"
)

type Rectangle struct {
	EmbedNode
	Size  image.Point
	Color *color.Color
	ctx   Context
}

func (r *Rectangle) Init(ctx Context) {
	*r = Rectangle{ctx: ctx}
	r.SetWrapper(r)
}
func (r *Rectangle) Measure(hint image.Point) image.Point {
	return r.Size
}
func (r *Rectangle) Paint() {
	if r.Color == nil {
		return
	}
	imageutil.FillRectangle(r.ctx.Image(), &r.Bounds, *r.Color)
}
