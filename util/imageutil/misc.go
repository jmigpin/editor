package imageutil

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
)

func DrawUniformMask(
	dst draw.Image,
	r *image.Rectangle,
	c color.Color,
	mask image.Image, mp image.Point,
	op draw.Op,
) {
	if c == nil {
		return
	}

	// improve performance for bgra (no difference if mask!=nil)
	if bgra, ok := dst.(*BGRA); ok {
		dst, c = bgra.RGBAImageWithCorrectedColor(c)
	}

	src := image.NewUniform(c)
	draw.DrawMask(dst, *r, src, image.Point{}, mask, mp, op)
}

func DrawUniform(dst draw.Image, r *image.Rectangle, c color.Color, op draw.Op) {
	DrawUniformMask(dst, r, c, nil, image.Point{}, op)
}

//----------

func DrawCopy(dst draw.Image, src image.Image, r *image.Rectangle) {
	draw.Draw(dst, *r, src, image.Point{}, draw.Src)
}

//----------

func FillRectangle(img draw.Image, r *image.Rectangle, c color.Color) {
	DrawUniform(img, r, c, draw.Src)
}

func BorderRectangle(img draw.Image, r *image.Rectangle, c color.Color, size int) {
	var sr [4]image.Rectangle
	// top
	sr[0] = *r
	sr[0].Max.Y = r.Min.Y + size
	// bottom
	sr[1] = *r
	sr[1].Min.Y = r.Max.Y - size
	// left
	sr[2] = *r
	sr[2].Max.X = r.Min.X + size
	sr[2].Min.Y = r.Min.Y + size
	sr[2].Max.Y = r.Max.Y - size
	// right
	sr[3] = *r
	sr[3].Min.X = r.Max.X - size
	sr[3].Min.Y = r.Min.Y + size
	sr[3].Max.Y = r.Max.Y - size

	for _, r2 := range sr {
		r2 = r2.Intersect(*r)
		DrawUniform(img, &r2, c, draw.Src)
	}
}

//----------

func MaxPoint(p1, p2 image.Point) image.Point {
	if p1.X < p2.X {
		p1.X = p2.X
	}
	if p1.Y < p2.Y {
		p1.Y = p2.Y
	}
	return p1
}
func MinPoint(p1, p2 image.Point) image.Point {
	if p1.X > p2.X {
		p1.X = p2.X
	}
	if p1.Y > p2.Y {
		p1.Y = p2.Y
	}
	return p1
}

//----------

func IntRGBA(u int) color.RGBA {
	v := u & 0xffffff
	r := uint8((v << 0) >> 16)
	g := uint8((v << 8) >> 16)
	b := uint8((v << 16) >> 16)
	return color.RGBA{r, g, b, 255}
}

func SprintRGB(c color.Color) string {
	rgba := convertToRGBAColor(c)
	return fmt.Sprintf("%x %x %x", rgba.R, rgba.G, rgba.B)
}

//----------

// maxColorDiff in [0.0, 1.0]
func PaintShadow(img draw.Image, r image.Rectangle, height int, maxColorDiff float64) {
	fn := func(w color.Color, v float64) color.Color {
		if c, ok := w.(color.RGBA); ok {
			return shade(c, v)
		}
		return w
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
