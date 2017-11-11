package loopers

import "github.com/jmigpin/editor/imageutil"

type FlashSelectionLooper struct {
	EmbedLooper
	strl      *StringLooper
	bgl       *BgLooper
	dl        *DrawLooper
	Selection *FlashSelectionIndexes
}

func NewFlashSelectionLooper(strl *StringLooper, bgl *BgLooper, dl *DrawLooper) *FlashSelectionLooper {
	return &FlashSelectionLooper{strl: strl, bgl: bgl, dl: dl}
}
func (lpr *FlashSelectionLooper) Loop(fn func() bool) {
	if lpr.Selection == nil {
		lpr.OuterLooper().Loop(fn)
		return
	}
	sl := lpr.Selection
	s, e := sl.Start, sl.End
	if s > e {
		s, e = e, s
	}
	lpr.OuterLooper().Loop(func() bool {
		if lpr.strl.Ri >= s && lpr.strl.Ri < e {
			p := lpr.Selection.Perc
			lpr.dl.Fg = imageutil.TintOrShade(lpr.dl.Fg, p)
			lpr.bgl.Bg = imageutil.TintOrShade(lpr.bgl.Bg, p)
		}
		return fn()
	})
}

type FlashSelectionIndexes struct {
	Perc       float64
	Start, End int
}
