package loopers

import (
	"image/color"

	"github.com/jmigpin/editor/imageutil"
)

type BgLooper struct {
	EmbedLooper
	strl         *StringLooper
	dl           *DrawLooper
	Bg           color.Color
	NoColorizeBg color.Color // colorize only if different from this
}

func MakeBgLooper(strl *StringLooper, dl *DrawLooper) BgLooper {
	return BgLooper{strl: strl, dl: dl}
}
func (lpr *BgLooper) Loop(fn func() bool) {
	lpr.OuterLooper().Loop(func() bool {
		if lpr.Bg != lpr.NoColorizeBg {
			pb := lpr.strl.PenBoundsForImage()
			dr := pb.Add(lpr.dl.Bounds.Min).Intersect(*lpr.dl.Bounds)
			imageutil.FillRectangle(lpr.dl.Image, &dr, lpr.Bg)
		}
		return fn()
	})
}
