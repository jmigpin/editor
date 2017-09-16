package drawutil2

import (
	"image"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

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
	if ru == '\t' || ru == '\n' || ru == '\r' {
		zero := image.Rect(0, 0, 0, 0)
		dr := zero
		mask := image.NewAlpha(zero)
		maskp = image.Point{}
		adv, ok := fr.GlyphAdvance(ru)
		return dr, mask, maskp, adv, ok
	}
	return fr.Face.Glyph(dot, ru)
}
func (fr *FaceRunes) GlyphAdvance(ru rune) (advance fixed.Int26_6, ok bool) {
	if ru == '\t' {
		adv, ok := fr.Face.GlyphAdvance(' ')
		return adv * 8, ok
	}

	if ru == '\n' || ru == '\r' {
		a, ok := fr.Face.GlyphAdvance(' ')
		return a / 2, ok
	}
	return fr.Face.GlyphAdvance(ru)
}
