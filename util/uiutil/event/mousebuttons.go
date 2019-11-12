package event

type MouseButton int32

const (
	ButtonNone MouseButton = iota
	ButtonLeft MouseButton = 1 << (iota - 1)
	ButtonMiddle
	ButtonRight
	ButtonWheelUp
	ButtonWheelDown
	ButtonWheelLeft
	ButtonWheelRight
	ButtonBackward
	ButtonForward
)

type MouseButtons int32

func (mb MouseButtons) Has(b MouseButton) bool {
	return int32(mb)&int32(b) > 0
}
func (mb MouseButtons) HasAny(bs MouseButtons) bool {
	return int32(mb)&int32(bs) > 0
}
func (mb MouseButtons) Is(b MouseButton) bool {
	return int32(mb) == int32(b)
}
