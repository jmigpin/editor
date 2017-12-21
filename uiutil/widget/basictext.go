package widget

import (
	"image"
	"image/color"

	"github.com/jmigpin/editor/drawutil2/simpledrawer"
)

type BasicText struct {
	EmbedNode
	Str   string
	Color *color.Color
	ctx   Context
}

func NewBasicText(ctx Context) *BasicText {
	bt := &BasicText{ctx: ctx}
	bt.SetWrapper(bt)
	return bt
}
func (bt *BasicText) Measure(hint image.Point) image.Point {
	return simpledrawer.Measure(bt.ctx.FontFace1(), bt.Str, hint)
}
func (bt *BasicText) Paint() {
	if bt.Color == nil {
		return
	}
	simpledrawer.Draw(bt.ctx.Image(), bt.ctx.FontFace1(), bt.Str, &bt.Bounds, *bt.Color)
}
