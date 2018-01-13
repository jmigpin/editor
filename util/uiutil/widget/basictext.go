package widget

import (
	"image"

	"github.com/jmigpin/editor/util/drawutil/hsdrawer"
	"github.com/jmigpin/editor/util/imageutil"
)

type BasicText struct {
	EmbedNode
	Str    string
	ctx    ImageContext
	Drawer hsdrawer.HSDrawer
}

func NewBasicText(ctx ImageContext) *BasicText {
	bt := &BasicText{ctx: ctx}
	return bt
}

func (bt *BasicText) Measure(hint image.Point) image.Point {
	bt.Drawer.Face = bt.Theme.Font().Face(nil)
	bt.Drawer.Str = bt.Str
	return bt.Drawer.Measure(hint)
}
func (bt *BasicText) CalcChildsBounds() {
	_ = bt.Measure(bt.Bounds.Size())
}
func (bt *BasicText) Paint() {
	c0 := bt.Theme.Palette().Normal.Bg
	imageutil.FillRectangle(bt.ctx.Image(), &bt.Bounds, c0)

	bt.Drawer.Fg = bt.Theme.Palette().Normal.Fg
	bt.Drawer.Draw(bt.ctx.Image(), &bt.Bounds)
}
