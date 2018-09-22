package widget

type Border struct {
	*Padder
}

func NewBorder(ctx ImageContext, child Node) *Border {
	b := &Border{Padder: NewPadder(ctx, child)}
	b.Padder.colorName = "border"
	return b
}
