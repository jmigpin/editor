package loopers

import "image/color"

type HighlightSegments struct {
	EmbedLooper
	strl *String
	bgl  *Bg
	dl   *Draw
	opt  *HighlightSegmentsOpt
}

func MakeHighlightSegments(strl *String, bgl *Bg, dl *Draw, opt *HighlightSegmentsOpt) HighlightSegments {
	return HighlightSegments{strl: strl, bgl: bgl, dl: dl, opt: opt}
}

func (lpr *HighlightSegments) Loop(fn func() bool) {
	segs := lpr.opt.OrderedSegments
	index := 0

	lpr.OuterLooper().Loop(func() bool {
		if lpr.strl.IsRiClone() {
			return fn()
		}

		colorize := false
		ri := lpr.strl.Ri
		for ; index < len(segs); index++ {
			e := segs[index]
			start, end := e[0], e[1]
			if ri < start {
				break
			}
			if ri < end {
				colorize = true
				break
			}
		}

		if colorize {
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

type HighlightSegmentsOpt struct {
	Fg, Bg          color.Color
	OrderedSegments [][2]int
}
