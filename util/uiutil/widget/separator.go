package widget

type Separator struct {
	Rectangle
}

func NewSeparator(ctx ImageContext) *Separator {
	return &Separator{Rectangle: *NewRectangle(ctx)}
}
func (s *Separator) Paint() {
	s.paint(s.TreeThemePaletteColor("fg"))
}
