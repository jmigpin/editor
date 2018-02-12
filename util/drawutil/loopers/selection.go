package loopers

import "image/color"

type Selection struct {
	EmbedLooper
	strl *String
	bgl  *Bg
	dl   *Draw
	opt  *SelectionOpt
}

func MakeSelection(strl *String, bgl *Bg, dl *Draw, opt *SelectionOpt) Selection {
	return Selection{strl: strl, bgl: bgl, dl: dl, opt: opt}
}
func (lpr *Selection) Loop(fn func() bool) {
	s, e := lpr.opt.Start, lpr.opt.End
	if s > e {
		s, e = e, s
	}

	lpr.OuterLooper().Loop(func() bool {
		if lpr.strl.IsRiClone() {
			return fn()
		}
		if lpr.strl.Ri >= s && lpr.strl.Ri < e {
			if lpr.opt.Fg != nil {
				lpr.dl.Fg = lpr.opt.Fg
			}
			if lpr.opt.Bg != nil {
				lpr.bgl.Bg = lpr.opt.Bg
			}
		}
		return fn()
	})
}

type SelectionOpt struct {
	Fg, Bg     color.Color
	Start, End int
}
