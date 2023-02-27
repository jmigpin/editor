package imageutil

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

// maxColorDiff in [0.0, 1.0]
func PaintShadow(img draw.Image, r image.Rectangle, height int, maxColorDiff float64) {
	fn := func(c color.Color, v float64) color.Color {
		//return Shade(c, v)
		// paint shadow only if already rgba (allows noticing if not using RGBA colors)
		if c2, ok := c.(color.RGBA); ok {
			return shade(c2, v)
		}
		return c
	}

	step := 0
	dy := float64(height)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		yperc := float64(step) / dy
		step++

		//v := maxColorDiff * (1 - yperc)

		// -(1/log(2))*log(x+1)+1
		u := -(1/math.Log(2))*math.Log(yperc+1) + 1
		v := maxColorDiff * u

		for x := r.Min.X; x < r.Max.X; x++ {
			atc := img.At(x, y)
			c := fn(atc, v)
			img.Set(x, y, c)
		}
	}
}
