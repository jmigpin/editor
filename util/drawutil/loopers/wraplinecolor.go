package loopers

import "image/color"

type WrapLineColor struct {
	EmbedLooper
	wlinel *WrapLine
	dl     *Draw
	bgl    *Bg
	Fg, Bg color.Color
}

func MakeWrapLineColor(wlinel *WrapLine, dl *Draw, bgl *Bg) WrapLineColor {
	return WrapLineColor{wlinel: wlinel, dl: dl, bgl: bgl}
}
func (lpr *WrapLineColor) Loop(fn func() bool) {
	lpr.OuterLooper().Loop(func() bool {
		switch lpr.wlinel.state {
		case 0: // other runes
		case 1, 2, 3: // wrapline: bg1, bg2, and rune
			lpr.bgl.Bg = lpr.Bg
			lpr.dl.Fg = lpr.Fg
		}
		return fn()
	})
}
