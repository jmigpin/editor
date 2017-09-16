package loopers

import "image/color"

type SelectionLooper struct {
	Looper    Looper
	strl      *StringLooper
	bgl       *BgLooper
	dl        *DrawLooper
	Selection *SelectionIndexes
	Fg, Bg    color.Color
}

func NewSelectionLooper(strl *StringLooper, bgl *BgLooper, dl *DrawLooper) *SelectionLooper {
	return &SelectionLooper{strl: strl, bgl: bgl, dl: dl}
}
func (lpr *SelectionLooper) Loop(fn func() bool) {
	lpr.Looper.Loop(func() bool {
		if lpr.colorize() {
			lpr.dl.Fg = lpr.Fg
			lpr.bgl.Bg = lpr.Bg
		}
		return fn()
	})
}
func (lpr *SelectionLooper) colorize() bool {
	sl := lpr.Selection
	if sl == nil {
		return false
	}
	if lpr.strl.RiClone {
		return false
	}
	s, e := sl.Start, sl.End
	if s > e {
		s, e = e, s
	}
	return lpr.strl.Ri >= s && lpr.strl.Ri < e
}

type SelectionIndexes struct {
	Start, End int
}
