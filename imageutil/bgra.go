package imageutil

import (
	"image"
	"image/color"
	"image/draw"
)

type BGRA struct {
	image.RGBA
}

func NewBGRA(r *image.Rectangle) *BGRA {
	u := image.NewRGBA(*r)
	return &BGRA{*u}
}

func NewBGRAFromBuffer(buf []byte, r *image.Rectangle) *BGRA {
	rgba := image.RGBA{Pix: buf, Stride: 4 * r.Dx(), Rect: *r}
	return &BGRA{RGBA: rgba}
}
func BGRASize(r *image.Rectangle) int {
	return r.Dx() * r.Dy() * 4
}

func (img *BGRA) ColorModel() color.Model {
	panic("!")
}
func (img *BGRA) Set(x, y int, c color.Color) {
	//u := color.RGBAModel.Convert(c).(color.RGBA) // slow
	u := convertToRGBAColor(c)
	img.SetRGBA(x, y, u)
}
func convertToRGBAColor(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	return color.RGBA{
		uint8(r >> 8),
		uint8(g >> 8),
		uint8(b >> 8),
		uint8(a >> 8),
	}
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
