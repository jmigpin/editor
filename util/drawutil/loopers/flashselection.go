package loopers

import (
	"image/color"

	"github.com/jmigpin/editor/util/imageutil"
)

type FlashSelection struct {
	EmbedLooper
	strl *String
	bgl  *Bg
	dl   *Draw
	opt  *FlashSelectionOpt
}

func MakeFlashSelection(strl *String, bgl *Bg, dl *Draw, opt *FlashSelectionOpt) FlashSelection {
	return FlashSelection{strl: strl, bgl: bgl, dl: dl, opt: opt}
}
func (lpr *FlashSelection) Loop(fn func() bool) {
	s, e := lpr.opt.Start, lpr.opt.End
	if s > e {
		s, e = e, s
	}
	lpr.OuterLooper().Loop(func() bool {
		// commented: flash wraplines and annotations
		//if lpr.strl.IsRiClone() {
		//	return fn()
		//}

		if lpr.strl.Ri >= s && lpr.strl.Ri < e {
			p := lpr.opt.Perc
			lpr.dl.Fg = imageutil.TintOrShade(lpr.dl.Fg, p)

			bg := lpr.bgl.Bg
			if bg == nil {
				bg = lpr.opt.Bg
			}
			if bg != nil {
				lpr.bgl.Bg = imageutil.TintOrShade(bg, p)
			}
		}
		return fn()
	})
}

type FlashSelectionOpt struct {
	Perc       float64
	Start, End int

	// Background to use if the bg has not been set yet by other extensions. This should be the textarea normal background color.
	Bg color.Color
}
