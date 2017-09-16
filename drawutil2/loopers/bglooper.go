package loopers

import (
	"image/color"

	"github.com/jmigpin/editor/imageutil"
)

type BgLooper struct {
	Looper Looper
	strl   *StringLooper
	dl     *DrawLooper
	Bg     color.Color
}

func NewBgLooper(strl *StringLooper, dl *DrawLooper) *BgLooper {
	return &BgLooper{strl: strl, dl: dl}
}
func (lpr *BgLooper) Loop(fn func() bool) {
	lpr.Looper.Loop(func() bool {
		if lpr.Bg != nil {
			pb := lpr.strl.PenBoundsForImage()
			dr := pb.Add(lpr.dl.Bounds.Min).Intersect(*lpr.dl.Bounds)
			imageutil.FillRectangle(lpr.dl.Image, &dr, lpr.Bg)
		}
		return fn()
	})
}
