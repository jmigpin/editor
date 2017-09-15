package imageutil

import "image/color"

func Tint(c0 color.Color, v float64) color.Color {
	c2 := color.RGBAModel.Convert(c0).(color.RGBA)
	c := c2
	c.R += uint8(v * float64((255 - c2.R)))
	c.G += uint8(v * float64((255 - c2.G)))
	c.B += uint8(v * float64((255 - c2.B)))
	return c
}

func Shade(c0 color.Color, v float64) color.RGBA {
	c2 := color.RGBAModel.Convert(c0).(color.RGBA)
	c := c2
	c.R = uint8(v * float64(c2.R))
	c.G = uint8(v * float64(c2.G))
	c.B = uint8(v * float64(c2.B))
	return c
}
