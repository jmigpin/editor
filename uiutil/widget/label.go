package widget

import (
	"image/color"

	"github.com/jmigpin/editor/imageutil"
)

type Label struct {
	ShellEmbedNode
	Text   BasicText
	Border Pad
	Pad    Pad
	Color  *color.Color
	ctx    Context
}

func (l *Label) Init(ctx Context) {
	*l = Label{}
	l.SetWrapper(l)
	l.ctx = ctx
	l.Text.Init(ctx)
	l.Pad.Init(ctx, &l.Text)
	l.Border.Init(ctx, &l.Pad)
	l.Append(&l.Border)
}
func (l *Label) Paint() {
	if l.Color == nil {
		return
	}
	u := l.Bounds()
	imageutil.FillRectangle(l.ctx.Image(), &u, *l.Color)
}
