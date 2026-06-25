package imageutil

import "image/color"

func EnsureContrastColor(fg, bg color.Color) color.Color {
	if fg == nil || bg == nil {
		return fg
	}
	fl := colorLuma8(fg)
	bl := colorLuma8(bg)
	if absInt(fl-bl) >= 125 {
		return fg
	}
	if bl >= 128 {
		return scaleColorRGB(fg, minFloat64(0.4, 0.72*(255.0/maxFloat64(1, float64(fl)))))
	}
	return tintColorRGB(fg, 0.55)
}

func colorLuma8(c color.Color) int {
	r, g, b, _ := c.RGBA()
	return int((299*r + 587*g + 114*b + 500) / 1000 >> 8)
}

func scaleColorRGB(c color.Color, factor float64) color.Color {
	r, g, b, a := c.RGBA()
	return color.RGBA{
		R: uint8(clampInt(int(float64(r>>8)*factor), 0, 255)),
		G: uint8(clampInt(int(float64(g>>8)*factor), 0, 255)),
		B: uint8(clampInt(int(float64(b>>8)*factor), 0, 255)),
		A: uint8(a >> 8),
	}
}

func tintColorRGB(c color.Color, amount float64) color.Color {
	r, g, b, a := c.RGBA()
	mix := func(v uint32) uint8 {
		u := float64(v >> 8)
		u += (255 - u) * amount
		return uint8(clampInt(int(u), 0, 255))
	}
	return color.RGBA{mix(r), mix(g), mix(b), uint8(a >> 8)}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
