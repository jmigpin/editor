package loopers

import "image/color"

type HighlightSegments struct {
	EmbedLooper
	strl  *String
	bgl   *Bg
	dl    *Draw
	index int
	opt   *HighlightSegmentsOpt
}

func MakeHighlightSegments(strl *String, bgl *Bg, dl *Draw, opt *HighlightSegmentsOpt) HighlightSegments {
	return HighlightSegments{strl: strl, bgl: bgl, dl: dl, opt: opt}
}

func (lpr *HighlightSegments) Loop(fn func() bool) {
	lpr.OuterLooper().Loop(func() bool {
		if lpr.strl.RiClone {
			return fn()
		}
		if lpr.colorize() {
			lpr.dl.Fg = lpr.opt.Fg
			lpr.bgl.Bg = lpr.opt.Bg
		}
		return fn()
	})
}
func (lpr *HighlightSegments) colorize() bool {
	segs := lpr.opt.OrderedSegments
	ri := lpr.strl.Ri
	for ; lpr.index < len(segs); lpr.index++ {
		e := segs[lpr.index]
		start, end := e[0], e[1]
		if ri < start {
			return false
		}
		if ri < end {
			return true
		}
	}
	return false
}

type HighlightSegmentsOpt struct {
	Fg, Bg          color.Color
	OrderedSegments [][2]int
}
