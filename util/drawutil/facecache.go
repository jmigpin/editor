package drawutil

import (
	"crypto/sha1"
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
	hru map[string]rune
}

func NewFaceCache(face font.Face) *FaceCache {
	fc := &FaceCache{Face: face}
	fc.gc = make(map[rune]*GlyphCache)
	fc.gac = make(map[rune]*GlyphAdvanceCache)
	fc.gbc = make(map[rune]*GlyphBoundsCache)
	fc.kc = make(map[string]fixed.Int26_6)
	fc.hru = make(map[string]rune)
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
		var zeroDot fixed.Point26_6 // always use zero
		dr, mask, maskp, adv, ok := fc.Face.Glyph(zeroDot, ru)

		// avoid the truetype package cache (it's not giving the same mask everytime, probably needs cache parameter)
		if ok {
			mask = copyMask(mask)

			//m, hash := copyMask2(mask)
			//hs := string(hash)
			//ru2, ok := fc.hru[hs]
			//if ok {
			//	//log.Printf("already exists: ru=%c exists in ru=%c", ru, ru2)
			//	gc, _ := fc.gc[ru2]
			//	mask = gc.mask
			//} else {
			//	fc.hru[hs] = ru
			//	mask = m
			//}
		}

		gc = &GlyphCache{dr, mask, maskp, adv, ok}
		fc.gc[ru] = gc
	}

	//p := image.Point{dot.X.Round(), dot.Y.Round()}
	p := image.Point{dot.X.Floor(), dot.Y.Floor()}
	dr2 := gc.dr.Add(p)

	return dr2, gc.mask, gc.maskp, gc.advance, gc.ok
}
func (fc *FaceCache) GlyphAdvance(ru rune) (advance fixed.Int26_6, ok bool) {
	gac, ok := fc.gac[ru]
	if !ok {
		adv, ok := fc.Face.GlyphAdvance(ru) // only one can run at a time
		gac = &GlyphAdvanceCache{adv, ok}
		fc.gac[ru] = gac
	}
	return gac.advance, gac.ok
}
func (fc *FaceCache) GlyphBounds(ru rune) (bounds fixed.Rectangle26_6, advance fixed.Int26_6, ok bool) {
	gbc, ok := fc.gbc[ru]
	if !ok {
		bounds, adv, ok := fc.Face.GlyphBounds(ru)
		gbc = &GlyphBoundsCache{bounds, adv, ok}
		fc.gbc[ru] = gbc
	}
	return gbc.bounds, gbc.advance, gbc.ok
}
func (fc *FaceCache) Kern(r0, r1 rune) fixed.Int26_6 {
	i := string([]rune{r0, r1})
	k, ok := fc.kc[i]
	if !ok {
		k = fc.Face.Kern(r0, r1) // only one can run at a time
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
type GlyphAdvanceCache struct {
	advance fixed.Int26_6
	ok      bool
}
type GlyphBoundsCache struct {
	bounds  fixed.Rectangle26_6
	advance fixed.Int26_6
	ok      bool
}

func copyMask(mask image.Image) image.Image {
	alpha := *(mask.(*image.Alpha)) // copy structure
	pix := make([]uint8, len(alpha.Pix))
	copy(pix, alpha.Pix)
	alpha.Pix = pix
	return &alpha
}

//----------

func copyMask2(mask image.Image) (image.Image, []byte) {
	alpha := *(mask.(*image.Alpha)) // copy structure
	pix := make([]uint8, len(alpha.Pix))
	copy(pix, alpha.Pix)
	alpha.Pix = pix
	h := bytesHash(pix)
	return &alpha, h
}

func bytesHash(b []byte) []byte {
	h := sha1.New()
	h.Write(b)
	return h.Sum(nil)
}
