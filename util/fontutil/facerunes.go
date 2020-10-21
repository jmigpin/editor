package fontutil

import (
	"image"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

var TabWidth = 8 // n times the space glyph
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
	ru2, adv, ok := fr.replace(ru)
	if ok {
		dr, mask, maskp, _, ok := fr.Face.Glyph(dot, ru2)
		return dr, mask, maskp, adv, ok
	}
	return fr.Face.Glyph(dot, ru)
}

func (fr *FaceRunes) GlyphBounds(ru rune) (bounds fixed.Rectangle26_6, advance fixed.Int26_6, ok bool) {
	ru2, adv, ok := fr.replace(ru)
	if ok {
		bounds, _, ok := fr.Face.GlyphBounds(ru2)
		return bounds, adv, ok
	}
	return fr.Face.GlyphBounds(ru)
}

func (fr *FaceRunes) GlyphAdvance(ru rune) (advance fixed.Int26_6, ok bool) {
	_, adv, ok := fr.replace(ru)
	if ok {
		return adv, ok
	}
	return fr.Face.GlyphAdvance(ru)
}

//----------

func (fr *FaceRunes) replace(ru0 rune) (rune, fixed.Int26_6, bool) {
	switch ru0 {
	case '\t':
		ru := ' '
		adv, ok := fr.Face.GlyphAdvance(ru)
		adv *= fixed.Int26_6(TabWidth)
		return ru, adv, ok
	case '\n':
		ru := ' '
		adv, ok := fr.Face.GlyphAdvance(ru)
		adv /= 2
		return ru, adv, ok
	case '\r':
		ru := CarriageReturnRune
		adv, ok := fr.Face.GlyphAdvance(ru)
		return ru, adv, ok
	case 0:
		ru := NullRune
		adv, ok := fr.Face.GlyphAdvance(ru)
		return ru, adv, ok
	case -1: // -1=eof
		ru := ' '
		return ru, 0, true
	}
	return 0, 0, false
}
