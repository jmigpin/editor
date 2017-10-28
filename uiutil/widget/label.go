package widget

import (
	"image/color"

	"github.com/jmigpin/editor/imageutil"
)

type Label struct {
	ShellEmbedNode
	Text   *BasicText
	Border *Border
	Pad    *Border
	Bg     color.Color
	ctx    Context
}

func NewLabel(ctx Context) *Label {
	l := &Label{}
	l.SetWrapper(l)
	l.ctx = ctx
	l.Bg = color.White
	l.Text = NewBasicText(ctx)
	l.Pad = NewBorder(ctx, l.Text)
	l.Border = NewBorder(ctx, l.Pad)
	l.Append(l.Border)
	return l
}
func (l *Label) Paint() {
	if l.Bg != nil {
		u := l.Bounds()
		imageutil.FillRectangle(l.ctx.Image(), &u, l.Bg)
	}
}
