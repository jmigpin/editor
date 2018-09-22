package widget

type Label struct {
	ENode
	Text   *Text
	Border *Border
	Pad    *Pad
	ctx    ImageContext
}

func NewLabel(ctx ImageContext) *Label {
	l := &Label{ctx: ctx}
	l.Text = NewText(ctx)
	l.Pad = NewPad(ctx, l.Text)
	l.Border = NewBorder(ctx, l.Pad)
	l.Append(l.Border)
	return l
}

//----------

func (l *Label) OnThemeChange() {
	bg := l.TreeThemePaletteColor("text_bg")
	// using l.SetThemePaletteColor() will lead to callback loop
	l.theme.SetPaletteColor("pad", bg)
}
