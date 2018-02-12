package widget

import (
	"image"

	"github.com/jmigpin/editor/util/drawutil/hsdrawer"
	"github.com/jmigpin/editor/util/imageutil"
)

type BasicText struct {
	EmbedNode
	ctx    ImageContext
	Drawer hsdrawer.HSDrawer
}

func NewBasicText(ctx ImageContext) *BasicText {
	return &BasicText{ctx: ctx}
}
func (bt *BasicText) SetStr(str string) {
	bt.Drawer.Args.Str = str
}
func (bt *BasicText) Measure(hint image.Point) image.Point {
	bt.Drawer.Args.Face = bt.Theme.Font().Face(nil)
	return bt.Drawer.Measure(hint)
}
func (bt *BasicText) CalcChildsBounds() {
	_ = bt.Measure(bt.Bounds.Size())
}
func (bt *BasicText) Paint() {
	bg := bt.Theme.Palette().Normal.Bg
	imageutil.FillRectangle(bt.ctx.Image(), &bt.Bounds, bg)
	bt.Drawer.Fg = bt.Theme.Palette().Normal.Fg
	bt.Drawer.Draw(bt.ctx.Image(), &bt.Bounds)
}
