package loopers

import "image/color"

type WrapLineColorLooper struct {
	EmbedLooper
	wlinel *WrapLineLooper
	dl     *DrawLooper
	bgl    *BgLooper
	Fg, Bg color.Color
}

func (lpr *WrapLineColorLooper) Init(wlinel *WrapLineLooper, dl *DrawLooper, bgl *BgLooper) {
	*lpr = WrapLineColorLooper{wlinel: wlinel, dl: dl, bgl: bgl}
}
func (lpr *WrapLineColorLooper) Loop(fn func() bool) {
	lpr.OuterLooper().Loop(func() bool {
		switch lpr.wlinel.state {
		case 1, 2: // bg1 and bg2
			lpr.bgl.Bg = lpr.Bg
		case 3: // wrapline rune
			lpr.dl.Fg = lpr.Fg
		}
		return fn()
	})
}
