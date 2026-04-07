package fontutil

import (
	"image"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

type FaceEmoji struct {
	font.Face
	emojiFace font.Face
}

func NewFaceEmoji(face, emojiFace font.Face) *FaceEmoji {
	return &FaceEmoji{Face: face, emojiFace: emojiFace}
}

func (fe *FaceEmoji) Glyph(dot fixed.Point26_6, ru rune) (
	dr image.Rectangle,
	mask image.Image,
	maskp image.Point,
	advance fixed.Int26_6,
	ok bool,
) {
	if face := fe.face(ru); face != nil {
		return face.Glyph(dot, ru)
	}
	return fe.Face.Glyph(dot, ru)
}

func (fe *FaceEmoji) GlyphBounds(ru rune) (bounds fixed.Rectangle26_6, advance fixed.Int26_6, ok bool) {
	if face := fe.face(ru); face != nil {
		return face.GlyphBounds(ru)
	}
	return fe.Face.GlyphBounds(ru)
}

func (fe *FaceEmoji) GlyphAdvance(ru rune) (advance fixed.Int26_6, ok bool) {
	if face := fe.face(ru); face != nil {
		return face.GlyphAdvance(ru)
	}
	return fe.Face.GlyphAdvance(ru)
}

func (fe *FaceEmoji) Kern(r0, r1 rune) fixed.Int26_6 {
	if fe.face(r0) != nil || fe.face(r1) != nil {
		return 0
	}
	return fe.Face.Kern(r0, r1)
}

//----------

func (fe *FaceEmoji) face(ru rune) font.Face {
	if _, ok := fe.Face.GlyphAdvance(ru); ok {
		return nil
	}
	adv, ok := fe.emojiFace.GlyphAdvance(ru)
	if !ok {
		return nil
	}
	if adv > fe.maxEmojiAdvance() {
		return nil
	}
	return fe.emojiFace
}

func (fe *FaceEmoji) maxEmojiAdvance() fixed.Int26_6 {
	adv, ok := fe.Face.GlyphAdvance('W')
	if ok {
		return adv
	}
	return fixed.I(2)
}
