package loopers

import (
	"image"
	"image/color"
	"image/draw"
)

type DrawLooper struct {
	Looper Looper
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

	lpr.Looper.Loop(func() bool {
		dr, mask, maskp, _, ok := strl.Face.Glyph(strl.Pen, strl.Ru)
		if !ok {
			return true
		}

		// clip
		dr = dr.Add(bounds.Min)
		dr2 := dr.Intersect(*bounds)
		if dr.Min.X < bounds.Min.X {
			maskp.X += bounds.Min.X - dr.Min.X
			//maskp.X += dr.Dx() - dr2.Dx()
		}
		if dr.Min.Y < bounds.Min.Y {
			maskp.Y += bounds.Min.Y - dr.Min.Y
			//maskp.Y += dr.Dy() - dr2.Dy()
		}

		ufg := image.NewUniform(lpr.Fg)
		draw.DrawMask(
			img, dr2,
			ufg, image.Point{},
			mask, maskp,
			draw.Over)

		return fn()
	})
}
