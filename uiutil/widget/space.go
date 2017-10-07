package widget

type Space struct {
	Rectangle
}

func NewSpace(ui UIer) *Space {
	s := &Space{}
	s.Rectangle.ui = ui
	return s
}
