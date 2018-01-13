package loopers

import (
	"image/color"

	"github.com/jmigpin/editor/util/imageutil"
)

type Bg struct {
	EmbedLooper
	strl *String
	dl   *Draw
	Bg   color.Color
}

func MakeBg(strl *String, dl *Draw) Bg {
	return Bg{strl: strl, dl: dl}
}
func (lpr *Bg) Loop(fn func() bool) {
	lpr.OuterLooper().Loop(func() bool {
		if lpr.Bg != nil {
			pb := lpr.strl.PenBoundsForImage()
			dr := pb.Add(lpr.dl.Bounds.Min).Intersect(*lpr.dl.Bounds)
			imageutil.FillRectangle(lpr.dl.Image, &dr, lpr.Bg)
		}
		return fn()
	})
}
