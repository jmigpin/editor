package widget

type Label struct {
	EmbedNode
	Text   *BasicText
	Border *Border
	Pad    *Pad
	ctx    ImageContext
}

func NewLabel(ctx ImageContext) *Label {
	l := &Label{ctx: ctx}
	l.Text = NewBasicText(ctx)
	l.Pad = NewPad(ctx, l.Text)
	l.Border = NewBorder(ctx, l.Pad)
	l.Append(l.Border)
	return l
}
