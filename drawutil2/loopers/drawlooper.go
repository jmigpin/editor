package loopers

import (
	"image"
	"image/color"
	"image/draw"
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
		// early exit if beyond max Y
		pb := strl.PenBounds()
		if pb.Max.Y.Floor() > bounds.Max.Y {
			return false
		}

		dr, mask, maskp, _, ok := strl.Face.Glyph(strl.Pen, strl.Ru)
		if !ok {
			return fn()
		}

		// clip
		dr = dr.Add(bounds.Min)
		dr2 := dr.Intersect(*bounds)
		if dr.Min.X < bounds.Min.X {
			maskp.X += bounds.Min.X - dr.Min.X
		}
		if dr.Min.Y < bounds.Min.Y {
			maskp.Y += bounds.Min.Y - dr.Min.Y
		}

		if lpr.Fg == nil {
			panic("fg is nil")
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
