package imageutil

import (
	"image/color"
	"sort"
)

// https://en.wikipedia.org/wiki/HSL_and_HSV#/media/File:HSV-RGB-comparison.svg

type HSV struct {
	H    uint16 // [0-360)
	S, V uint8  // [0-255]
}

// h in [0-360), {s,v} in [0-100]
func MakeHSV(h uint16, s, v uint8) HSV {
	return HSV{
		H: h,
		S: uint8(uint32(s) * 255 / 100),
		V: uint8(uint32(v) * 255 / 100),
	}
}

func (c HSV) RGBA() (_, _, _, _ uint32) {
	r, g, b := hsv2rgb(c.H, c.S, c.V)
	rgba := color.RGBA{r, g, b, 0xff}
	return rgba.RGBA()
}

//----------

var HSVModel color.Model = color.ModelFunc(convertToHSV)

func convertToHSV(c color.Color) color.Color {
	c2 := color.RGBAModel.Convert(c).(color.RGBA)
	h, s, v := rgb2hsv(c2.R, c2.G, c2.B)
	return HSV{h, s, v}
}

//----------
//----------
//----------

func hsv2rgb(h uint16, s, v uint8) (r, g, b uint8) {
	max0 := int(v)
	min0 := int(v) * (255 - int(s)) / 255

	offset := int(h % 60)
	mid0 := (max0 - min0) * offset / 60

	max := uint8(max0)
	min := uint8(min0)
	mid := uint8(mid0)

	seg := h / 60
	switch seg {
	case 0:
		return max, min + mid, min
	case 1:
		return max - mid, max, min
	case 2:
		return min, max, min + mid
	case 3:
		return min, max - mid, max
	case 4:
		return min + mid, min, max
	case 5:
		return max, min, max - mid
	default:
		panic("!")
	}
}

func rgb2hsv(r0, g0, b0 uint8) (h uint16, s, v uint8) {
	r, g, b := int(r0), int(g0), int(b0)

	u := []int{r, g, b}
	sort.Ints(u)
	m1, _, m3 := u[0], u[1], u[2]

	max := m3
	max0 := max
	v = uint8(max0)

	min := m1
	min0 := min
	s = uint8(0)
	if v != 0 {
		s = uint8(-(min0*255/int(v) - 255))
	}

	seg := 0
	mid := 0
	if b == m1 {
		if r == m3 {
			seg = 0
			mid = g - min
		} else {
			seg = 1
			mid = max - r
		}
	} else if r == m1 {
		if g == m3 {
			seg = 2
			mid = b - min
		} else {
			seg = 3
			mid = max - g
		}
	} else {
		if b == m3 {
			seg = 4
			mid = r - min
		} else {
			seg = 5
			mid = max - b
		}
	}

	mid0 := mid
	offset := 0
	d := max0 - min0
	if d != 0 {
		offset = mid0 * 60 / d
	}
	h = uint16(seg*60 + offset)

	return
}

//----------
//----------
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
