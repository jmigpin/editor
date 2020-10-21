package fontutil

import (
	"image"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

type FaceCache struct {
	font.Face
	gc  map[rune]*GlyphCache
	gac map[rune]*GlyphAdvanceCache
	gbc map[rune]*GlyphBoundsCache
	kc  map[string]fixed.Int26_6 // kern cache
}

func NewFaceCache(face font.Face) *FaceCache {
	fc := &FaceCache{Face: face}
	fc.gc = make(map[rune]*GlyphCache)
	fc.gac = make(map[rune]*GlyphAdvanceCache)
	fc.gbc = make(map[rune]*GlyphBoundsCache)
	fc.kc = make(map[string]fixed.Int26_6)
	return fc
}
func (fc *FaceCache) Glyph(dot fixed.Point26_6, ru rune) (
	dr image.Rectangle,
	mask image.Image,
	maskp image.Point,
	advance fixed.Int26_6,
	ok bool,
) {
	gc, ok := fc.gc[ru]
	if !ok {
		gc = NewGlyphCache(fc.Face, ru)
		fc.gc[ru] = gc
	}
	p := image.Point{dot.X.Floor(), dot.Y.Floor()}
	dr2 := gc.dr.Add(p)
	return dr2, gc.mask, gc.maskp, gc.advance, gc.ok
}
func (fc *FaceCache) GlyphAdvance(ru rune) (advance fixed.Int26_6, ok bool) {
	gac, ok := fc.gac[ru]
	if !ok {
		gac = NewGlyphAdvanceCache(fc.Face, ru)
		fc.gac[ru] = gac
	}
	return gac.advance, gac.ok
}
func (fc *FaceCache) GlyphBounds(ru rune) (bounds fixed.Rectangle26_6, advance fixed.Int26_6, ok bool) {
	gbc, ok := fc.gbc[ru]
	if !ok {
		gbc = NewGlyphBoundsCache(fc.Face, ru)
		fc.gbc[ru] = gbc
	}
	return gbc.bounds, gbc.advance, gbc.ok
}
func (fc *FaceCache) Kern(r0, r1 rune) fixed.Int26_6 {
	i := kernIndex(r0, r1)
	k, ok := fc.kc[i]
	if !ok {
		k = NewKernCache(fc.Face, r0, r1)
		fc.kc[i] = k
	}
	return k
}

//----------

type GlyphCache struct {
	dr      image.Rectangle
	mask    image.Image
	maskp   image.Point
	advance fixed.Int26_6
	ok      bool
}

func NewGlyphCache(face font.Face, ru rune) *GlyphCache {
	var zeroDot fixed.Point26_6 // always use zero
	dr, mask, maskp, adv, ok := face.Glyph(zeroDot, ru)

	// avoid the truetype package cache (it's not giving the same mask everytime, probably needs cache parameter)
	if ok {
		mask = copyMask(mask)
	}

	return &GlyphCache{dr, mask, maskp, adv, ok}
}

//----------

type GlyphAdvanceCache struct {
	advance fixed.Int26_6
	ok      bool
}

func NewGlyphAdvanceCache(face font.Face, ru rune) *GlyphAdvanceCache {
	adv, ok := face.GlyphAdvance(ru) // only one can run at a time
	return &GlyphAdvanceCache{adv, ok}
}

//----------

type GlyphBoundsCache struct {
	bounds  fixed.Rectangle26_6
	advance fixed.Int26_6
	ok      bool
}

func NewGlyphBoundsCache(face font.Face, ru rune) *GlyphBoundsCache {
	bounds, adv, ok := face.GlyphBounds(ru) // only one can run at a time
	return &GlyphBoundsCache{bounds, adv, ok}
}

//----------

func kernIndex(r0, r1 rune) string {
	return string([]rune{r0, ',', r1})
}

func NewKernCache(face font.Face, r0, r1 rune) fixed.Int26_6 {
	return face.Kern(r0, r1) // only one can run at a time
}

//----------

func copyMask(mask image.Image) image.Image {
	alpha := *(mask.(*image.Alpha)) // copy structure
	pix := make([]uint8, len(alpha.Pix))
	copy(pix, alpha.Pix)
	alpha.Pix = pix
	return &alpha
}

//----------

//func copyMask2(mask image.Image) (image.Image, []byte) {
//	alpha := *(mask.(*image.Alpha)) // copy structure
//	pix := make([]uint8, len(alpha.Pix))
//	copy(pix, alpha.Pix)
//	alpha.Pix = pix
//	h := bytesHash(pix)
//	return &alpha, h
//}

//func bytesHash(b []byte) []byte {
//	h := sha1.New()
//	h.Write(b)
//	return h.Sum(nil)
//}
