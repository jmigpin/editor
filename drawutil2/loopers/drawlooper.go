package loopers

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/jmigpin/editor/imageutil"
)

type DrawLooper struct {
	EmbedLooper
	strl   *StringLooper
	Fg     color.Color
	Image  draw.Image
	Bounds *image.Rectangle
}

func NewDrawLooper(strl *StringLooper, image draw.Image, bounds *image.Rectangle) *DrawLooper {
	return &DrawLooper{strl: strl, Image: image, Bounds: bounds}
}
func (lpr *DrawLooper) Loop(fn func() bool) {
	bounds := lpr.Bounds
	strl := lpr.strl
	img := lpr.Image

	lpr.OuterLooper().Loop(func() bool {
		// allow to skip draw with a rune 0
		if strl.Ru == 0 {
			return fn()
		}

		dr, mask, maskp, _, ok := strl.Face.Glyph(strl.Pen, strl.Ru)
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
