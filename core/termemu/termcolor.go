package termemu

import "image/color"

type TermColor struct {
	kind  TermColorKind
	index uint8
	rgba  color.RGBA
}

func NewTermColorIndexed(n int) TermColor {
	return TermColor{kind: TermColorIndexed, index: uint8(n)}
}

func NewTermColorRGB(r, g, b uint8) TermColor {
	return TermColor{kind: TermColorRGB, rgba: color.RGBA{r, g, b, 255}}
}

func (tc TermColor) IsDefault() bool {
	return tc.kind == TermColorDefault
}

func (tc TermColor) Kind() TermColorKind {
	return tc.kind
}

func (tc TermColor) Index() int {
	return int(tc.index)
}

func (tc TermColor) RGBA() color.RGBA {
	return tc.rgba
}

//----------

type TermColorKind uint8

const (
	TermColorDefault TermColorKind = iota
	TermColorIndexed
	TermColorRGB
)

//----------

func XTerm256Color(n int) color.Color {
	switch {
	case 0 <= n && n <= 15:
		ansi16 := [16]color.RGBA{
			{0, 0, 0, 255},       // 0
			{205, 0, 0, 255},     // 1
			{0, 205, 0, 255},     // 2
			{205, 205, 0, 255},   // 3
			{0, 0, 238, 255},     // 4
			{205, 0, 205, 255},   // 5
			{0, 205, 205, 255},   // 6
			{229, 229, 229, 255}, // 7
			{127, 127, 127, 255}, // 8
			{255, 0, 0, 255},     // 9
			{0, 255, 0, 255},     // 10
			{255, 255, 0, 255},   // 11
			{92, 92, 255, 255},   // 12
			{255, 0, 255, 255},   // 13
			{0, 255, 255, 255},   // 14
			{255, 255, 255, 255}, // 15
		}
		return ansi16[n]
	case 16 <= n && n <= 231:
		k := n - 16
		levels := [6]uint8{0, 95, 135, 175, 215, 255}
		r := levels[k/36]
		g := levels[(k/6)%6]
		b := levels[k%6]
		return color.RGBA{r, g, b, 255}
	case 232 <= n && n <= 255:
		v := uint8(8 + (n-232)*10)
		return color.RGBA{v, v, v, 255}
	default:
		panic("!")
	}
}
