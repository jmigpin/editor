package drawutil

import (
	"image"

	"golang.org/x/image/math/fixed"
)

type FaceRunes struct {
	*FaceCache
}

func NewFaceRunes(face *FaceCache) *FaceRunes {
	return &FaceRunes{face}
}

func (fr *FaceRunes) Glyph(ru rune) (dr image.Rectangle, mask image.Image, maskp image.Point, advance fixed.Int26_6, ok bool) {
	if ru == '\t' || ru == '\n' || ru == eofRune {
		zero := image.Rect(0, 0, 0, 0)
		dr := zero
		mask := image.NewRGBA(zero)
		maskp = image.Point{}
		adv, ok := fr.FaceCache.GlyphAdvance(ru)
		return dr, mask, maskp, adv, ok
	}
	return fr.FaceCache.Glyph(ru)
}

func (fr *FaceRunes) GlyphAdvance(ru rune) (advance fixed.Int26_6, ok bool) {
	if ru == '\t' {
		adv, ok := fr.FaceCache.GlyphAdvance(' ')
		return adv * 8, ok
	}
	if ru == '\n' || ru == eofRune {
		return fr.FaceCache.GlyphAdvance(' ')
	}
	return fr.FaceCache.GlyphAdvance(ru)
}
