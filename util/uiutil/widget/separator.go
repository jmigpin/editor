package widget

type Separator struct {
	Rectangle
	Handle *SeparatorHandle
}

func NewSeparator(ctx ImageContext, ml *MultiLayer) *Separator {
	s := &Separator{
		Rectangle: *NewRectangle(ctx),
	}
	s.Handle = NewSeparatorHandle(s)
	ml.SeparatorLayer.Append(s.Handle)
	return s
}

func (s *Separator) Close() {
	// remove handle from multilayer
	s.Handle.Parent.Remove(s.Handle)
}

func (s *Separator) Layout() {
	s.Rectangle.Layout()
	s.Handle.Layout()
}
