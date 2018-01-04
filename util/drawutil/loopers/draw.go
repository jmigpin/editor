package loopers

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/jmigpin/editor/util/imageutil"
)

type Draw struct {
	EmbedLooper
	strl   *String
	Fg     color.Color
	Image  draw.Image
	Bounds *image.Rectangle
}

func MakeDraw(strl *String, image draw.Image, bounds *image.Rectangle) Draw {
	return Draw{strl: strl, Image: image, Bounds: bounds}
}
func (lpr *Draw) Loop(fn func() bool) {
	bounds := lpr.Bounds
	strl := lpr.strl
	img := lpr.Image

	lpr.OuterLooper().Loop(func() bool {
		// allow to skip draw with a rune 0
		if strl.Ru == 0 {
			return fn()
		}

		baselinePen := strl.Pen
		baselinePen.Y += strl.Baseline()
		dr, mask, maskp, _, ok := strl.Face.Glyph(baselinePen, strl.Ru)
		if !ok {
			return fn()
		}

		// clip
		dr = dr.Add(bounds.Min)
		if dr.Min.X < bounds.Min.X {
			maskp.X += bounds.Min.X - dr.Min.X
		}
		if dr.Min.Y < bounds.Min.Y {
			maskp.Y += bounds.Min.Y - dr.Min.Y
		}

		if lpr.Fg == nil {
			panic("fg is nil")
		}

		dr2 := dr.Intersect(*bounds)
		imageutil.DrawUniformMask(img, &dr2, lpr.Fg, mask, maskp, draw.Over)

		return fn()
	})
}
