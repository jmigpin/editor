package drawutil

import (
	"image"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Same as FaceCache but with locks.
type FaceCacheL struct {
	font.Face
	mu sync.RWMutex
	gc     map[rune]*GlyphCache
	gac    map[rune]*GlyphAdvanceCache
	gbc    map[rune]*GlyphBoundsCache
	kc     map[string]fixed.Int26_6 // kern cache
}

func NewFaceCacheL(face font.Face) *FaceCacheL {
	fc := &FaceCacheL{Face: face}
	fc.gc = make(map[rune]*GlyphCache)
	fc.gac = make(map[rune]*GlyphAdvanceCache)
	fc.gbc = make(map[rune]*GlyphBoundsCache)
	fc.kc = make(map[string]fixed.Int26_6)
	return fc
}
func (fc *FaceCacheL) Glyph(dot fixed.Point26_6, ru rune) (
	dr image.Rectangle,
	mask image.Image,
	maskp image.Point,
	advance fixed.Int26_6,
	ok bool,
) {
	fc.mu.RLock()
	gc, ok := fc.gc[ru]
	fc.mu.RUnlock()
	if !ok {
		fc.mu.Lock()

		var zeroDot fixed.Point26_6 // always use dot zero
		dr, mask, maskp, adv, ok := fc.Face.Glyph(zeroDot, ru)

		// avoid the truetype package cache (it's not giving the same mask everytime, probably needs cache parameter)
		if ok {
			mask = copyMask(mask)
		}

		gc = &GlyphCache{dr, mask, maskp, adv, ok}
		fc.gc[ru] = gc

		fc.mu.Unlock()
	}

	//p := image.Point{dot.X.Round(), dot.Y.Round()}
	p := image.Point{dot.X.Floor(), dot.Y.Floor()}
	dr2 := gc.dr.Add(p)

	return dr2, gc.mask, gc.maskp, gc.advance, gc.ok
}
func (fc *FaceCacheL) GlyphAdvance(ru rune) (advance fixed.Int26_6, ok bool) {
	fc.mu.RLock()
	gac, ok := fc.gac[ru]
	fc.mu.RUnlock()
	if !ok {
		fc.mu.Lock()
		adv, ok := fc.Face.GlyphAdvance(ru) // only one can run at a time
		gac = &GlyphAdvanceCache{adv, ok}
		fc.gac[ru] = gac
		fc.mu.Unlock()
	}
	return gac.advance, gac.ok
}
func (fc *FaceCacheL) GlyphBounds(ru rune) (bounds fixed.Rectangle26_6, advance fixed.Int26_6, ok bool) {
	fc.mu.RLock()
	gbc, ok := fc.gbc[ru]
	fc.mu.RUnlock()
	if !ok {
		fc.mu.Lock()
		bounds, adv, ok := fc.Face.GlyphBounds(ru)
		gbc = &GlyphBoundsCache{bounds, adv, ok}
		fc.gbc[ru] = gbc
		fc.mu.Unlock()
	}
	return gbc.bounds, gbc.advance, gbc.ok
}
func (fc *FaceCacheL) Kern(r0, r1 rune) fixed.Int26_6 {
	i := string([]rune{r0, r1})
	fc.mu.RLock()
	k, ok := fc.kc[i]
	fc.mu.RUnlock()
	if !ok {
		fc.mu.Lock()
		k = fc.Face.Kern(r0, r1) // only one can run at a time
		fc.kc[i] = k
		fc.mu.Unlock()
	}
	return k
}
