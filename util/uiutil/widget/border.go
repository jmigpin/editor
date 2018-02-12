package widget

import "image/color"

type Border struct {
	*Pad
	fg color.Color
}

func NewBorder(ctx ImageContext, child Node) *Border {
	return &Border{Pad: NewPad(ctx, child)}
}
func (b *Border) Paint() {
	b.paint(b.Theme.Palette().Normal.Fg)
}
