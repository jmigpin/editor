package imageutil

import (
	"image"
	"image/color"
	"image/draw"
)

func DrawMask(
	dst draw.Image,
	r image.Rectangle,
	src image.Image, srcp image.Point,
	mask image.Image, maskp image.Point,
	op draw.Op,
) {
	// improve performance for bgra
	if bgra, ok := dst.(*BGRA); ok {
		dst = &bgra.RGBA
	}

	draw.DrawMask(dst, r, src, srcp, mask, maskp, op)
}

//----------

func DrawUniformMask(
	dst draw.Image,
	r image.Rectangle,
	c color.Color,
	mask image.Image, maskp image.Point,
	op draw.Op,
) {
	if c == nil {
		return
	}
	// correct color for bgra
	if _, ok := dst.(*BGRA); ok {
		c = BgraColor(c)
	}

	src := image.NewUniform(c)
	srcp := image.ZP
	DrawMask(dst, r, src, srcp, mask, maskp, op)
}

func DrawUniform(dst draw.Image, r image.Rectangle, c color.Color, op draw.Op) {
	DrawUniformMask(dst, r, c, nil, image.ZP, op)
}

//----------

func DrawCopy(dst draw.Image, r image.Rectangle, src image.Image) {
	DrawMask(dst, r, src, image.ZP, nil, image.ZP, draw.Src)
}

//----------

func FillRectangle(img draw.Image, r image.Rectangle, c color.Color) {
	DrawUniform(img, r, c, draw.Src)
}

func BorderRectangle(img draw.Image, r image.Rectangle, c color.Color, size int) {
	var sr [4]image.Rectangle
	// top
	sr[0] = r
	sr[0].Max.Y = r.Min.Y + size
	// bottom
	sr[1] = r
	sr[1].Min.Y = r.Max.Y - size
	// left
	sr[2] = r
	sr[2].Max.X = r.Min.X + size
	sr[2].Min.Y = r.Min.Y + size
	sr[2].Max.Y = r.Max.Y - size
	// right
	sr[3] = r
	sr[3].Min.X = r.Max.X - size
	sr[3].Min.Y = r.Min.Y + size
	sr[3].Max.Y = r.Max.Y - size

	for _, r2 := range sr {
		r2 = r2.Intersect(r)
		DrawUniform(img, r2, c, draw.Src)
	}
}
