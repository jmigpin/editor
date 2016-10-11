package xutil

import (
	"image"
	"image/color"
	"image/draw"
)

type BGRA struct {
	RGBA image.RGBA
}

func (img *BGRA) ColorModel() color.Model {
	panic("!")
}
func (img *BGRA) Bounds() image.Rectangle {
	return img.RGBA.Bounds()
}
func (img *BGRA) Set(x, y int, c color.Color) {
	//u := color.RGBAModel.Convert(c).(color.RGBA) // slow
	u := convertToRGBA(c)
	img.SetRGBA(x, y, u)
}

// Allows fast lane if detected.
func (img *BGRA) SetRGBA(x, y int, c color.RGBA) {
	c.R, c.B = c.B, c.R
	img.RGBA.SetRGBA(x, y, c)
}
func (img *BGRA) At(x, y int) color.Color {
	c := img.RGBA.RGBAAt(x, y)
	c.R, c.B = c.B, c.R
	return c
}
func (img *BGRA) SubImage(r image.Rectangle) draw.Image {
	u := img.RGBA.SubImage(r).(*image.RGBA)
	return &BGRA{*u}
}

func convertToRGBA(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	return color.RGBA{
		uint8(r >> 8),
		uint8(g >> 8),
		uint8(b >> 8),
		uint8(a >> 8),
	}
}
