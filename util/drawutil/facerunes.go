package drawutil

import (
	"image"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

var TabWidth = 8
var CarriageReturnRune = rune(8453) // "care of" symbol but using it for "carriage return"

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
	switch ru {
	case '\t', '\n':
		zero := image.Rect(0, 0, 0, 0)
		dr := zero
		mask := image.NewAlpha(zero)
		maskp = image.Point{}
		adv, ok := fr.GlyphAdvance(ru)
		return dr, mask, maskp, adv, ok
	case '\r':
		ru = CarriageReturnRune
	}
	return fr.Face.Glyph(dot, ru)
}
func (fr *FaceRunes) GlyphAdvance(ru rune) (advance fixed.Int26_6, ok bool) {
	switch ru {
	case '\t':
		adv, ok := fr.Face.GlyphAdvance(' ')
		return adv * fixed.Int26_6(TabWidth), ok
	case '\n':
		a, ok := fr.Face.GlyphAdvance(' ')
		return a / 2, ok
	case '\r':
		ru = CarriageReturnRune
	}
	return fr.Face.GlyphAdvance(ru)
}
