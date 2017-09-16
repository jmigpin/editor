package loopers

import (
	"image"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// glyph metrics
// https://developer.apple.com/library/content/documentation/TextFonts/Conceptual/CocoaTextArchitecture/Art/glyph_metrics_2x.png

type StringLooper struct {
	EmbedLooper // not used, this is the outmost looper

	Face    font.Face
	Str     string
	Ri      int
	Ru      rune
	PrevRu  rune
	Pen     fixed.Point26_6
	Advance fixed.Int26_6
	Metrics *font.Metrics

	// set externally - helps detect extra drawn runes (ex: wraplinerune)
	RiClone bool
}

func NewStringLooper(face font.Face, str string) *StringLooper {
	fm := face.Metrics()
	lpr := &StringLooper{
		Face:    face,
		Metrics: &fm,
		Str:     str,
	}
	lpr.Pen.Y = lpr.LineBaseline()
	return lpr
}
func (lpr *StringLooper) Loop(fn func() bool) {
	o := lpr.Ri
	for Ri, Ru := range lpr.Str[o:] {
		lpr.Ri = o + Ri
		lpr.Ru = Ru
		if !lpr.Iterate(fn) {
			return
		}
	}
	// set ri to allow testing that it reached the end
	lpr.Ri = len(lpr.Str)
}
func (lpr *StringLooper) Iterate(fn func() bool) bool {
	lpr.AddKern()
	lpr.CalcAdvance()
	if ok := fn(); !ok {
		return false
	}
	lpr.PrevRu = lpr.Ru
	lpr.Pen.X = lpr.PenXAdvance()
	return true
}
func (lpr *StringLooper) AddKern() {
	lpr.Pen.X += lpr.Face.Kern(lpr.PrevRu, lpr.Ru)
}
func (lpr *StringLooper) CalcAdvance() bool {
	adv, ok := lpr.Face.GlyphAdvance(lpr.Ru)
	if !ok {
		lpr.Advance = 0
		return false
	}
	lpr.Advance = adv
	return true
}
func (lpr *StringLooper) PenXAdvance() fixed.Int26_6 {
	return lpr.Pen.X + lpr.Advance
}
func (lpr *StringLooper) PenBounds() *fixed.Rectangle26_6 {
	var r fixed.Rectangle26_6
	r.Min.X = lpr.Pen.X
	r.Max.X = lpr.PenXAdvance()
	r.Min.Y = lpr.LineY0()
	r.Max.Y = lpr.LineY1()
	return &r
}
func (lpr *StringLooper) PenBoundsForImage() *image.Rectangle {
	pb := lpr.PenBounds()
	min := image.Point{pb.Min.X.Floor(), pb.Min.Y.Floor()}
	max := image.Point{pb.Max.X.Ceil(), pb.Max.Y.Ceil()}
	r := image.Rect(min.X, min.Y, max.X, max.Y)
	return &r
}

func (lpr *StringLooper) LineBaseline() fixed.Int26_6 {
	return lpr.Metrics.Ascent
}
func (lpr *StringLooper) LineHeight() fixed.Int26_6 {
	lh := lpr.LineBaseline() + lpr.Metrics.Descent
	return fixed.I(lh.Ceil())
}
func (lpr *StringLooper) LineY0() fixed.Int26_6 {
	return lpr.Pen.Y - lpr.LineBaseline()
}
func (lpr *StringLooper) LineY1() fixed.Int26_6 {
	return lpr.LineY0() + lpr.LineHeight()
}
