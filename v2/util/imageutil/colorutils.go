package imageutil

import (
	"fmt"
	"image/color"
)

func RgbaColor(c color.Color) color.RGBA {
	if u, ok := c.(color.RGBA); ok {
		return u
	} else {
		return convertToRgbaColor(c)
	}
}
func convertToRgbaColor(c color.Color) color.RGBA {
	// slow
	//return color.RGBAModel.Convert(c).(color.RGBA)

	r, g, b, a := c.RGBA()
	return color.RGBA{
		uint8(r >> 8),
		uint8(g >> 8),
		uint8(b >> 8),
		uint8(a >> 8),
	}
}

//----------

func RgbaFromInt(u int) color.RGBA {
	v := u & 0xffffff
	r := uint8((v << 0) >> 16)
	g := uint8((v << 8) >> 16)
	b := uint8((v << 16) >> 16)
	return color.RGBA{r, g, b, 255}
}

//----------

// Ex. usage: editor.xutil.cursors
func ColorUint16s(c color.Color) (uint16, uint16, uint16, uint16) {
	r, g, b, a := c.RGBA()
	return uint16(r << 8), uint16(g << 8), uint16(b << 8), uint16(a)
}

//----------

func SprintRgb(c color.Color) string {
	rgba := RgbaColor(c)
	return fmt.Sprintf("%x %x %x", rgba.R, rgba.G, rgba.B)
}

//----------

// Turn color lighter by v percent (0.0, 1.0).
func Tint(c color.Color, v float64) color.Color {
	c2 := RgbaColor(c)
	return tint(c2, v)
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

//----------

// Turn color darker by v percent (0.0, 1.0).
func Shade(c color.Color, v float64) color.Color {
	c2 := RgbaColor(c)
	return shade(c2, v)
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

//----------

func TintOrShade(c color.Color, v float64) color.Color {
	c2 := RgbaColor(c)
	if isLighter(c2) {
		return shade(c2, v)
	} else {
		return tint(c2, v)
	}
}

func IsLighter(c color.Color) bool {
	c2 := RgbaColor(c)
	return isLighter(c2)
}

func isLighter(c color.RGBA) bool {
	u := int(c.R) + int(c.G) + int(c.B)
	return u > 256*3/2
}

//----------

func Valorize(c color.Color, v float64, auto bool) color.Color {
	if v < -1 || v > 1 {
		panic("!")
	}
	hsv := HSVModel.Convert(c).(HSV)

	var u int = int(hsv.V)

	//d := int(float64(hsv.V) * v)
	d := int(255 * v)
	if auto {
		// auto decide to add or subtract
		if hsv.V < 255/2 {
			u += d
		} else {
			u -= d
		}
	} else {
		u += d
	}

	if u > 255 {
		hsv.V = 255
	} else if u < 0 {
		hsv.V = 0
	} else {
		hsv.V = uint8(u)
	}
	c2 := RgbaColor(hsv)
	return c2
}
