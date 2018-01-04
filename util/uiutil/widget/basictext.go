package widget

import (
	"image"
	"image/color"

	"github.com/jmigpin/editor/util/drawutil/hsdrawer"
)

type BasicText struct {
	EmbedNode
	Str    string
	Color  *color.Color
	ctx    Context
	Drawer *hsdrawer.HSDrawer
}

func NewBasicText(ctx Context) *BasicText {
	bt := &BasicText{ctx: ctx}
	bt.Drawer = hsdrawer.NewHSDrawer(bt.ctx.FontFace1())
	return bt
}
func (bt *BasicText) Measure(hint image.Point) image.Point {
	bt.Drawer.Face = bt.ctx.FontFace1() // suppport font change
	bt.Drawer.Str = bt.Str
	return bt.Drawer.Measure(hint)
}
func (bt *BasicText) CalcChildsBounds() {
	_ = bt.Measure(bt.Bounds.Size())
}
func (bt *BasicText) Paint() {
	if bt.Color == nil {
		return
	}

	// TODO: a measure needs to be done before calling the draw (needs to be ensured)

	bt.Drawer.Colors = &hsdrawer.Colors{Normal: hsdrawer.FgBg{*bt.Color, nil}}
	bt.Drawer.Draw(bt.ctx.Image(), &bt.Bounds)
}
