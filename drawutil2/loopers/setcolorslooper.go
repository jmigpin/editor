package loopers

import "image/color"

type SetColorsLooper struct {
	Looper Looper
	dl     *DrawLooper
	bgl    *BgLooper
	Fg, Bg color.Color
}

func NewSetColorsLooper(dl *DrawLooper, bgl *BgLooper) *SetColorsLooper {
	return &SetColorsLooper{dl: dl, bgl: bgl}
}
func (lpr *SetColorsLooper) Loop(fn func() bool) {
	lpr.Looper.Loop(func() bool {
		lpr.dl.Fg = lpr.Fg
		lpr.bgl.Bg = lpr.Bg
		return fn()
	})
}
