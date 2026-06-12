package fontutil

import (
	"image"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	"golang.org/x/text/width"
)

type FaceFallback struct {
	font.Face
	fallbackFaces []font.Face

	IsMono  bool
	MonoAdv fixed.Int26_6

	maxHeight fixed.Int26_6

	cache   map[rune]font.Face
	cachemu sync.Mutex
}

func NewFaceFallback(face font.Face, fallbackFaces []font.Face, isMono bool, monoAdv fixed.Int26_6) *FaceFallback {
	maxHeight, _ := faceLineHeightBaseline(face)

	return &FaceFallback{
		Face:          face,
		fallbackFaces: fallbackFaces,
		IsMono:        isMono,
		MonoAdv:       monoAdv,
		maxHeight:     maxHeight,
		cache:         make(map[rune]font.Face),
	}
}

func (ff *FaceFallback) Glyph(dot fixed.Point26_6, ru rune) (
	dr image.Rectangle,
	mask image.Image,
	maskp image.Point,
	advance fixed.Int26_6,
	ok bool,
) {
	if face := ff.face(ru); face != nil {
		dr, mask, maskp, advance, ok = face.Glyph(dot, ru)
	} else {
		dr, mask, maskp, advance, ok = ff.Face.Glyph(dot, ru)
	}
	if ff.IsMono {
		advance = ff.MonoAdv * fixed.Int26_6(runeMonoWidthMult(ru))
	}
	return
}

func (ff *FaceFallback) GlyphBounds(ru rune) (bounds fixed.Rectangle26_6, advance fixed.Int26_6, ok bool) {
	if face := ff.face(ru); face != nil {
		bounds, advance, ok = face.GlyphBounds(ru)
	} else {
		bounds, advance, ok = ff.Face.GlyphBounds(ru)
	}
	if ff.IsMono {
		advance = ff.MonoAdv * fixed.Int26_6(runeMonoWidthMult(ru))
	}
	return
}

func (ff *FaceFallback) GlyphAdvance(ru rune) (advance fixed.Int26_6, ok bool) {
	if face := ff.face(ru); face != nil {
		advance, ok = face.GlyphAdvance(ru)
	} else {
		advance, ok = ff.Face.GlyphAdvance(ru)
	}
	if ff.IsMono {
		advance = ff.MonoAdv * fixed.Int26_6(runeMonoWidthMult(ru))
	}
	return
}

func (ff *FaceFallback) Kern(r0, r1 rune) fixed.Int26_6 {
	if ff.face(r0) != nil || ff.face(r1) != nil {
		return 0
	}
	return ff.Face.Kern(r0, r1)
}

//----------

func (ff *FaceFallback) face(ru rune) font.Face {
	// check if main face has it
	if _, ok := ff.Face.GlyphAdvance(ru); ok {
		return nil
	}

	// check cache
	ff.cachemu.Lock()
	f, ok := ff.cache[ru]
	ff.cachemu.Unlock()
	if ok {
		return f
	}

	// find face by testing scales (starts at 1.0)
	for _, f2 := range ff.fallbackFaces {
		bounds, adv, ok := f2.GlyphBounds(ru)
		if !ok {
			continue
		}

		// check height
		h := bounds.Max.Y - bounds.Min.Y
		if h > ff.maxHeight {
			continue
		}

		// check width (only for mono)
		maxAdv := ff.MonoAdv * fixed.Int26_6(runeMonoWidthMult(ru))
		// Allow up to 40% width overflow to prevent wide/emoji runes from shrinking
		if ff.IsMono && adv > maxAdv*140/100 {
			continue
		}

		f = f2
		break
	}

	// update cache
	ff.cachemu.Lock()
	ff.cache[ru] = f
	ff.cachemu.Unlock()

	return f
}

//----------

func NewFallbackFaces(fallbackFont *Font, fopts FaceOptions) []font.Face {
	scales := []float64{1.0, 0.95, 0.9, 0.85, 0.8, 0.75, 0.7, 0.65, 0.6, 0.55, 0.5}
	faces := make([]font.Face, len(scales))
	for i, p := range scales {
		fopts2 := fopts
		fopts2.SetSize(fopts.Size() * p)
		faces[i] = mustNewFace(fallbackFont.Font, &fopts2.opts)
	}
	return faces
}

//----------

func runeMonoWidthMult(ru rune) int {
	p := width.LookupRune(ru)
	k := p.Kind()
	if k == width.EastAsianWide || k == width.EastAsianFullwidth {
		return 2
	}
	return 1
}
