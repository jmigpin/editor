package drawutil

import (
	"image"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

type FaceCache struct {
	Face   font.Face
	faceMu sync.RWMutex
	gac    map[rune]*GlyphAdvanceCache
	gc     map[rune]*GlyphCache
	kc     map[string]fixed.Int26_6 // kern cache
}

func NewFaceCache(face font.Face) *FaceCache {
	fc := &FaceCache{Face: face}
	fc.gac = make(map[rune]*GlyphAdvanceCache)
	fc.gc = make(map[rune]*GlyphCache)
	fc.kc = make(map[string]fixed.Int26_6)
	return fc
}

func (fc *FaceCache) Glyph(ru rune) (dr image.Rectangle, mask image.Image, maskp image.Point, advance fixed.Int26_6, ok bool) {
	fc.faceMu.RLock()
	gc, ok := fc.gc[ru]
	fc.faceMu.RUnlock()
	if !ok {
		fc.faceMu.Lock()

		var dot0 fixed.Point26_6 // always use dot zero
		dr, mask, maskp, adv, ok := fc.Face.Glyph(dot0, ru)

		// avoid the truetype package cache (it's not giving the same mask everytime, probably needs cache parameter)
		if ok {
			mask = copyMask(mask)
		}

		gc = &GlyphCache{dr, mask, maskp, adv, ok}
		fc.gc[ru] = gc

		fc.faceMu.Unlock()
	}
	return gc.dr, gc.mask, gc.maskp, gc.advance, gc.ok
}
func (fc *FaceCache) GlyphAdvance(ru rune) (advance fixed.Int26_6, ok bool) {
	fc.faceMu.RLock()
	gac, ok := fc.gac[ru]
	fc.faceMu.RUnlock()
	if !ok {
		fc.faceMu.Lock()
		adv, ok := fc.Face.GlyphAdvance(ru) // only one can run at a time
		gac = &GlyphAdvanceCache{adv, ok}
		fc.gac[ru] = gac
		fc.faceMu.Unlock()
	}
	return gac.advance, gac.ok
}
func (fc *FaceCache) Kern(r0, r1 rune) fixed.Int26_6 {
	i := string(r0) + string(r1)
	fc.faceMu.RLock()
	k, ok := fc.kc[i]
	fc.faceMu.RUnlock()
	if !ok {
		fc.faceMu.Lock()
		k = fc.Face.Kern(r0, r1) // only one can run at a time
		fc.kc[i] = k
		fc.faceMu.Unlock()
	}
	return k
}

type GlyphAdvanceCache struct {
	advance fixed.Int26_6
	ok      bool
}

type GlyphCache struct {
	dr      image.Rectangle
	mask    image.Image
	maskp   image.Point
	advance fixed.Int26_6
	ok      bool
}

func copyMask(mask image.Image) image.Image {
	alpha := *(mask.(*image.Alpha))
	pix := make([]uint8, len(alpha.Pix))
	copy(pix, alpha.Pix)
	alpha.Pix = pix
	return &alpha
}
