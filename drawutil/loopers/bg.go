package loopers

import (
	"image/color"

	"github.com/jmigpin/editor/imageutil"
)

type Bg struct {
	EmbedLooper
	strl         *String
	dl           *Draw
	Bg           color.Color
	NoColorizeBg color.Color // colorize only if different from this
}

func MakeBg(strl *String, dl *Draw) Bg {
	return Bg{strl: strl, dl: dl}
}
func (lpr *Bg) Loop(fn func() bool) {
	lpr.OuterLooper().Loop(func() bool {
		if lpr.Bg != lpr.NoColorizeBg {
			pb := lpr.strl.PenBoundsForImage()
			dr := pb.Add(lpr.dl.Bounds.Min).Intersect(*lpr.dl.Bounds)
			imageutil.FillRectangle(lpr.dl.Image, &dr, lpr.Bg)
		}
		return fn()
	})
}
