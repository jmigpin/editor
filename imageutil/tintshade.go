package imageutil

import "image/color"

// Turn color lighter by v percent (0.0, 1.0).
func Tint(c0 color.Color, v float64) color.Color {
	if v < 0 || v > 1 {
		panic("!")
	}
	c2 := color.RGBAModel.Convert(c0).(color.RGBA)
	c := c2
	c.R += uint8(v * float64((255 - c2.R)))
	c.G += uint8(v * float64((255 - c2.G)))
	c.B += uint8(v * float64((255 - c2.B)))
	return c
}

// Turn color darker by v percent (0.0, 1.0).
func Shade(c0 color.Color, v float64) color.RGBA {
	if v < 0 || v > 1 {
		panic("!")
	}
	v = 1.0 - v
	c2 := color.RGBAModel.Convert(c0).(color.RGBA)
	c := c2
	c.R = uint8(v * float64(c2.R))
	c.G = uint8(v * float64(c2.G))
	c.B = uint8(v * float64(c2.B))
	return c
}
