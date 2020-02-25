package drawutil

import (
	"image"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Same as FaceCacheL but with sync.map.
type FaceCacheL2 struct {
	font.Face
	mu  sync.RWMutex
	gc  sync.Map
	gac sync.Map
	gbc sync.Map
	kc  sync.Map // kern cache
}

func NewFaceCacheL2(face font.Face) *FaceCacheL2 {
	fc := &FaceCacheL2{Face: face}
	return fc
}
func (fc *FaceCacheL2) Glyph(dot fixed.Point26_6, ru rune) (
	dr image.Rectangle,
	mask image.Image,
	maskp image.Point,
	advance fixed.Int26_6,
	ok bool,
) {
	v, ok := fc.gc.Load(ru)
	var gc *GlyphCache
	if ok {
		gc = v.(*GlyphCache)
	} else {
		fc.mu.Lock()
		gc = NewGlyphCache(fc.Face, ru)
		fc.gc.Store(ru, gc)
		fc.mu.Unlock()
	}
	p := image.Point{dot.X.Floor(), dot.Y.Floor()}
	dr2 := gc.dr.Add(p)
	return dr2, gc.mask, gc.maskp, gc.advance, gc.ok
}
func (fc *FaceCacheL2) GlyphAdvance(ru rune) (advance fixed.Int26_6, ok bool) {
	v, ok := fc.gac.Load(ru)
	var gac *GlyphAdvanceCache
	if ok {
		gac = v.(*GlyphAdvanceCache)
	} else {
		fc.mu.Lock()
		gac = NewGlyphAdvanceCache(fc.Face, ru)
		fc.gac.Store(ru, gac)
		fc.mu.Unlock()
	}
	return gac.advance, gac.ok
}
func (fc *FaceCacheL2) GlyphBounds(ru rune) (bounds fixed.Rectangle26_6, advance fixed.Int26_6, ok bool) {
	v, ok := fc.gbc.Load(ru)
	var gbc *GlyphBoundsCache
	if ok {
		gbc = v.(*GlyphBoundsCache)
	} else {
		fc.mu.Lock()
		gbc = NewGlyphBoundsCache(fc.Face, ru)
		fc.gbc.Store(ru, gbc)
		fc.mu.Unlock()
	}
	return gbc.bounds, gbc.advance, gbc.ok
}
func (fc *FaceCacheL2) Kern(r0, r1 rune) fixed.Int26_6 {
	i := kernIndex(r0, r1)
	v, ok := fc.kc.Load(i)
	var k fixed.Int26_6
	if ok {
		k = v.(fixed.Int26_6)
	} else {
		fc.mu.Lock()
		k = NewKernCache(fc.Face, r0, r1)
		fc.kc.Store(i, k)
		fc.mu.Unlock()
	}
	return k
}
