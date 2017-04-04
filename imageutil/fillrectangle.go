package imageutil

import (
	"image"
	"image/color"
	"image/draw"
)

func FillRectangle(img0 draw.Image, r *image.Rectangle, c0 color.Color) {
	img, c := RGBAImageAndColor(img0, c0)
	u := image.NewUniform(c)
	draw.Draw(img, *r, u, image.Point{}, draw.Src)
}

func RGBAImageAndColor(img draw.Image, c color.Color) (*image.RGBA, *color.RGBA) {
	c2 := color.RGBAModel.Convert(c).(color.RGBA)
	// BGRA
	bgra, ok := img.(*BGRA)
	if ok {
		c2.R, c2.B = c2.B, c2.R
		return &bgra.RGBA, &c2
	}
	// RGBA
	rgba := img.(*image.RGBA)
	return rgba, &c2
}
