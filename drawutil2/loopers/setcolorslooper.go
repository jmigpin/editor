package loopers

import "image/color"

type SetColorsLooper struct {
	EmbedLooper
	dl     *DrawLooper
	bgl    *BgLooper
	Fg, Bg color.Color
}

func NewSetColorsLooper(dl *DrawLooper, bgl *BgLooper) *SetColorsLooper {
	return &SetColorsLooper{dl: dl, bgl: bgl}
}
func (lpr *SetColorsLooper) Loop(fn func() bool) {
	// also set if the string is empty, other loopers might need the colors
	lpr.dl.Fg = lpr.Fg
	lpr.bgl.Bg = lpr.Bg

	lpr.OuterLooper().Loop(func() bool {
		lpr.dl.Fg = lpr.Fg
		lpr.bgl.Bg = lpr.Bg
		return fn()
	})
}
