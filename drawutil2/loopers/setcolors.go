package loopers

import "image/color"

type SetColors struct {
	EmbedLooper
	dl     *Draw
	bgl    *Bg
	Fg, Bg color.Color
}

func NewSetColors(dl *Draw, bgl *Bg) *SetColors {
	return &SetColors{dl: dl, bgl: bgl}
}
func (lpr *SetColors) Loop(fn func() bool) {
	// also set if the string is empty, other loopers might need the colors
	lpr.dl.Fg = lpr.Fg
	lpr.bgl.Bg = lpr.Bg

	lpr.OuterLooper().Loop(func() bool {
		lpr.dl.Fg = lpr.Fg
		lpr.bgl.Bg = lpr.Bg
		return fn()
	})
}
