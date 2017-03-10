package drawutil

import (
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

const eofRune = rune(0)

type StringIterator struct {
	face *Face

	fm *font.Metrics

	str         string
	ri          int
	ru, prevRu  rune
	pen, penEnd fixed.Point26_6
}

func NewStringIterator(face *Face, str string) *StringIterator {
	fm := face.Face.Metrics()
	iter := &StringIterator{
		face: face,
		fm:   &fm,
		str:  str,
	}
	iter.pen.Y = LineBaseline(iter.fm)
	return iter
}
func (iter *StringIterator) Loop(fn func() bool) {
	o := iter.ri
	for ri, ru := range iter.str[o:] {
		iter.ri = o + ri
		iter.ru = ru
		if !iter.iterate(fn) {
			return
		}
	}

	// end-of-file
	iter.ri = len(iter.str)
	iter.ru = eofRune
	_ = iter.iterate(fn)
}
func (iter *StringIterator) iterate(fn func() bool) bool {
	iter.addKernToPen()
	iter.calcPenEnd()
	if ok := fn(); !ok {
		return false
	}
	iter.prevRu = iter.ru
	iter.pen = iter.penEnd
	return true
}
func (iter *StringIterator) addKernToPen() {
	iter.pen.X += iter.face.Kern(iter.prevRu, iter.ru)
}
func (iter *StringIterator) calcPenEnd() bool {
	adv, ok := iter.face.GlyphAdvance(iter.ru)
	if !ok {
		return false
	}
	iter.penEnd = iter.pen
	iter.penEnd.X += adv
	return true
}
func (iter *StringIterator) PenBounds() *fixed.Rectangle26_6 {
	var r fixed.Rectangle26_6
	r.Min.X = iter.pen.X
	r.Max.X = iter.penEnd.X
	r.Min.Y = LineY0(iter.pen.Y, iter.fm)
	r.Max.Y = LineY1(iter.pen.Y, iter.fm)
	return &r
}
func (iter *StringIterator) LookaheadRune(n int) (rune, bool) {
	u := iter.str[iter.ri:]
	for i, ru := range u {
		if i >= n {
			return ru, true
		}
	}
	return rune(0), false
}
