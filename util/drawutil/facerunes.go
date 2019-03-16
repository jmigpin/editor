package drawutil

import (
	"image"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

var TabWidth = 8
var CarriageReturnRune = '♪'
var NullRune = '◦'

// Special runes face
type FaceRunes struct {
	font.Face
}

func NewFaceRunes(face font.Face) *FaceRunes {
	return &FaceRunes{face}
}

func (fr *FaceRunes) Glyph(dot fixed.Point26_6, ru rune) (
	dr image.Rectangle,
	mask image.Image,
	maskp image.Point,
	advance fixed.Int26_6,
	ok bool,
) {
	if ru < 0 { // -1=eof
		return image.ZR, nil, image.ZP, 0, false
	}
	switch ru {
	case '\t', '\n':
		return image.ZR, nil, image.ZP, 0, false
	case '\r':
		ru = CarriageReturnRune
	case 0:
		ru = NullRune
	}
	return fr.Face.Glyph(dot, ru)
}
func (fr *FaceRunes) GlyphAdvance(ru rune) (advance fixed.Int26_6, ok bool) {
	if ru < 0 { // -1=eof
		return fixed.Int26_6(0), false
	}
	switch ru {
	case '\t':
		adv, ok := fr.Face.GlyphAdvance(' ')
		return adv * fixed.Int26_6(TabWidth), ok
	case '\n':
		a, ok := fr.Face.GlyphAdvance(' ')
		return a / 2, ok
	case '\r':
		ru = CarriageReturnRune
	case 0:
		ru = NullRune
	}
	return fr.Face.GlyphAdvance(ru)
}
