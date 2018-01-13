package loopers

import "image/color"

type WrapLineColor struct {
	EmbedLooper
	wlinel *WrapLine
	dl     *Draw
	bgl    *Bg
	opt    *WrapLineColorOpt
}

func MakeWrapLineColor(wlinel *WrapLine, dl *Draw, bgl *Bg, opt *WrapLineColorOpt) WrapLineColor {
	return WrapLineColor{wlinel: wlinel, dl: dl, bgl: bgl, opt: opt}
}
func (lpr *WrapLineColor) Loop(fn func() bool) {
	lpr.OuterLooper().Loop(func() bool {
		switch lpr.wlinel.state {
		case 0: // other runes
		case 1, 2, 3: // wrapline: bg1, bg2, and rune
			lpr.bgl.Bg = lpr.opt.Bg
			lpr.dl.Fg = lpr.opt.Fg
		}
		return fn()
	})
}

type WrapLineColorOpt struct {
	Fg, Bg color.Color
}
