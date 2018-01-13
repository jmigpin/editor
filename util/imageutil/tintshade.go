package imageutil

import (
	"image/color"
)

func IsLighter(c color.Color) bool {
	c2 := color.RGBAModel.Convert(c).(color.RGBA)
	return isLighter(c2)
}

// Turn color lighter by v percent (0.0, 1.0).
func Tint(c color.Color, v float64) color.Color {
	c2 := color.RGBAModel.Convert(c).(color.RGBA)
	return tint(c2, v)
}

// Turn color darker by v percent (0.0, 1.0).
func Shade(c color.Color, v float64) color.Color {
	c2 := color.RGBAModel.Convert(c).(color.RGBA)
	return shade(c2, v)
}

func TintOrShade(c color.Color, v float64) color.Color {
	c2 := color.RGBAModel.Convert(c).(color.RGBA)
	if isLighter(c2) {
		return shade(c2, v)
	} else {
		return tint(c2, v)
	}
}

func isLighter(c color.RGBA) bool {
	u := int(c.R) + int(c.G) + int(c.B)
	return u > 256*3/2
}
func tint(c color.RGBA, v float64) color.Color {
	if v < 0 || v > 1 {
		panic("!")
	}
	c.R += uint8(v * float64((255 - c.R)))
	c.G += uint8(v * float64((255 - c.G)))
	c.B += uint8(v * float64((255 - c.B)))
	return c
}
func shade(c color.RGBA, v float64) color.Color {
	if v < 0 || v > 1 {
		panic("!")
	}
	v = 1.0 - v
	c.R = uint8(v * float64(c.R))
	c.G = uint8(v * float64(c.G))
	c.B = uint8(v * float64(c.B))
	return c
}
