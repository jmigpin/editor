package widget

import (
	"image"
	"image/color"

	"github.com/jmigpin/editor/imageutil"
)

type Rectangle struct {
	LeafEmbedNode
	Color color.Color
	Size  image.Point
	ctx   Context
}

func NewRectangle(ctx Context) *Rectangle {
	r := &Rectangle{
		ctx:  ctx,
		Size: image.Point{10, 10}, // this size is used in tests
	}
	r.SetWrapper(r)
	return r
}
func (r *Rectangle) Measure(max image.Point) image.Point {
	return r.Size
}
func (r *Rectangle) Paint() {
	if r.Color != nil {
		b := r.Bounds()
		imageutil.FillRectangle(r.ctx.Image(), &b, r.Color)
	}
}
