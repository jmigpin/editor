package fontutil

import (
	"image"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

type FaceFallback struct {
	font.Face
	fallbackFace font.Face
}

func NewFaceFallback(face, fallbackFace font.Face) *FaceFallback {
	return &FaceFallback{Face: face, fallbackFace: fallbackFace}
}

func (ff *FaceFallback) Glyph(dot fixed.Point26_6, ru rune) (
	dr image.Rectangle,
	mask image.Image,
	maskp image.Point,
	advance fixed.Int26_6,
	ok bool,
) {
	if face := ff.face(ru); face != nil {
		return face.Glyph(dot, ru)
	}
	return ff.Face.Glyph(dot, ru)
}

func (ff *FaceFallback) GlyphBounds(ru rune) (bounds fixed.Rectangle26_6, advance fixed.Int26_6, ok bool) {
	if face := ff.face(ru); face != nil {
		return face.GlyphBounds(ru)
	}
	return ff.Face.GlyphBounds(ru)
}

func (ff *FaceFallback) GlyphAdvance(ru rune) (advance fixed.Int26_6, ok bool) {
	if face := ff.face(ru); face != nil {
		return face.GlyphAdvance(ru)
	}
	return ff.Face.GlyphAdvance(ru)
}

func (ff *FaceFallback) Kern(r0, r1 rune) fixed.Int26_6 {
	if ff.face(r0) != nil || ff.face(r1) != nil {
		return 0
	}
	return ff.Face.Kern(r0, r1)
}

//----------

func (ff *FaceFallback) face(ru rune) font.Face {
	if _, ok := ff.Face.GlyphAdvance(ru); ok {
		return nil
	}
	adv, ok := ff.fallbackFace.GlyphAdvance(ru)
	if !ok {
		return nil
	}
	if adv > ff.maxFallbackAdvance() {
		return nil
	}
	return ff.fallbackFace
}

func (ff *FaceFallback) maxFallbackAdvance() fixed.Int26_6 {
	adv, ok := ff.Face.GlyphAdvance('W')
	if ok {
		return adv
	}
	return fixed.I(2)
}

//----------

func newFallbackFace(face font.Face, fallbackFont *Font, fopts FaceOptions, sampleRunes []rune) font.Face {
	fallbackFace := mustNewFace(fallbackFont.Font, &fopts.opts)
	maxAdv, ok := face.GlyphAdvance('W')
	if !ok {
		maxAdv = fixed.I(2)
	}
	maxHeight, _ := faceLineHeightBaseline(face)

	for _, p := range []float64{1.0, 0.95, 0.9, 0.85, 0.8, 0.75, 0.7, 0.65, 0.6, 0.55, 0.5, 0.45, 0.4} {
		fopts2 := fopts
		fopts2.SetSize(fopts.Size() * p)
		face2 := mustNewFace(fallbackFont.Font, &fopts2.opts)
		if fallbackFaceFits(face2, maxAdv, maxHeight, sampleRunes) {
			return face2
		}
	}
	return fallbackFace
}

func fallbackFaceFits(fallbackFace font.Face, maxAdv, maxHeight fixed.Int26_6, sampleRunes []rune) bool {
	for _, ru := range sampleRunes {
		bounds, adv, ok := fallbackFace.GlyphBounds(ru)
		if !ok {
			continue
		}
		if adv > maxAdv {
			return false
		}
		h := bounds.Max.Y - bounds.Min.Y
		if h > maxHeight {
			return false
		}
	}
	return true
}
