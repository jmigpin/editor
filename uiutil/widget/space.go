package widget

type Space struct {
	*Rectangle
}

func NewSpace(ctx Context) *Space {
	return &Space{Rectangle: NewRectangle(ctx)}
}
