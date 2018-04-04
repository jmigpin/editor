package widget

type Border struct {
	*Pad
}

func NewBorder(ctx ImageContext, child Node) *Border {
	return &Border{Pad: NewPad(ctx, child)}
}
func (b *Border) Paint() {
	b.paint(b.Theme.Palette().Get("fg"))
}
