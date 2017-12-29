package loopers

import (
	"image"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// glyph metrics
// https://developer.apple.com/library/content/documentation/TextFonts/Conceptual/CocoaTextArchitecture/Art/glyph_metrics_2x.png

type StringLooper struct {
	EmbedLooper

	Face    font.Face
	Str     string
	Ri      int
	Ru      rune
	PrevRu  rune
	Pen     fixed.Point26_6 // upper left corner
	Kern    fixed.Int26_6
	Advance fixed.Int26_6

	// set externally - helps detect extra drawn runes (ex: wraplinerune)
	RiClone bool

	Metrics font.Metrics
}

func MakeStringLooper(face font.Face, str string) StringLooper {
	return StringLooper{
		Face:    face,
		Str:     str,
		Metrics: face.Metrics(),
	}
}
func (lpr *StringLooper) Loop(fn func() bool) {
	lpr.OuterLooper().Loop(func() bool {
		if lpr.Ri >= len(lpr.Str) {
			return false
		}
		ru, w := utf8.DecodeRuneInString(lpr.Str[lpr.Ri:])
		lpr.Ru = ru
		lpr.RiClone = false
		if !lpr.Iterate(fn) {
			return false
		}
		lpr.Ri += w
		return true
	})
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
	lpr.Kern = lpr.Face.Kern(lpr.PrevRu, lpr.Ru)
	lpr.Pen.X += lpr.Kern
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

	// both min and max should use the same function (floor/ceil/round) because while the first rune uses ceil on max, if the next rune uses floor on min it will overwrite the previous rune on one pixel. This is noticeable in painting backgrounds.
	min := image.Point{pb.Min.X.Round(), pb.Min.Y.Round()}
	max := image.Point{pb.Max.X.Round(), pb.Max.Y.Round()}

	r := image.Rect(min.X, min.Y, max.X, max.Y)
	return &r
}

func (lpr *StringLooper) Baseline() fixed.Int26_6 {
	return lpr.Metrics.Ascent
}
func (lpr *StringLooper) LineHeight() fixed.Int26_6 {
	lh := lpr.Baseline() + lpr.Metrics.Descent
	// line height needs to be aligned with an int to have predictable line positions to be used in calculations.
	return fixed.I(lh.Ceil())
}
func (lpr *StringLooper) LineY0() fixed.Int26_6 {
	return lpr.Pen.Y
}
func (lpr *StringLooper) LineY1() fixed.Int26_6 {
	return lpr.LineY0() + lpr.LineHeight()
}

// Implements PosDataKeeper
func (lpr *StringLooper) KeepPosData() interface{} {
	d := &StringLooperData{
		Ri:     lpr.Ri,
		PrevRu: lpr.PrevRu,
		Pen:    lpr.Pen,
	}
	return d
}

// Implements PosDataKeeper
func (lpr *StringLooper) RestorePosData(data interface{}) {
	d := data.(*StringLooperData)
	lpr.Ri = d.Ri
	lpr.PrevRu = d.PrevRu
	lpr.Pen = d.Pen
}

// Implements PosDataKeeper
func (lpr *StringLooper) UpdatePosData() {
}

type StringLooperData struct {
	Ri     int
	PrevRu rune
	Pen    fixed.Point26_6
}
