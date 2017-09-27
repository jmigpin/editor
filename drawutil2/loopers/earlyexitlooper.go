package loopers

import "image"

type EarlyExitLooper struct {
	EmbedLooper
	strl   *StringLooper
	bounds *image.Rectangle
}

func NewEarlyExitLooper(strl *StringLooper, bounds *image.Rectangle) *EarlyExitLooper {
	return &EarlyExitLooper{strl: strl, bounds: bounds}
}
func (lpr *EarlyExitLooper) Loop(fn func() bool) {
	lpr.OuterLooper().Loop(func() bool {
		// early exit if beyond max Y
		pb := lpr.strl.PenBounds()
		if pb.Max.Y.Floor() > lpr.bounds.Max.Y {
			return false
		}
		return fn()
	})
}
