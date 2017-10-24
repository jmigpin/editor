package loopers

import (
	"image/color"
	"unicode"

	"golang.org/x/image/math/fixed"
)

type WrapLine2Looper struct {
	EmbedLooper
	strl  *StringLooper
	linei *LineLooper
	MaxX  fixed.Int26_6

	WrapData WrapLine2Data

	state int
}

func NewWrapLine2Looper(strl *StringLooper, linei *LineLooper, maxX fixed.Int26_6) *WrapLine2Looper {
	return &WrapLine2Looper{strl: strl, linei: linei, MaxX: maxX}
}
func (lpr *WrapLine2Looper) Loop(fn func() bool) {
	lpr.OuterLooper().Loop(func() bool {
		lpr.state = 0
		penXAdv := lpr.strl.PenXAdvance()

		// keep track of indentation for wrapped lines
		if !lpr.WrapData.NotStartingSpaces {
			if unicode.IsSpace(lpr.strl.Ru) {
				lpr.WrapData.PenX = penXAdv
			} else {
				lpr.WrapData.NotStartingSpaces = true
			}
		}

		// wrap line
		if lpr.strl.Ri > 0 && penXAdv > lpr.MaxX {
			runeAdv := penXAdv - lpr.strl.Pen.X
			runeCut := penXAdv - lpr.MaxX

			origRu := lpr.strl.Ru

			// bg close to the border - current rune size covers the space
			lpr.state = 1
			lpr.strl.RiClone = true
			lpr.strl.Ru = 0
			if ok := fn(); !ok {
				return false
			}
			lpr.strl.RiClone = false

			// newline
			lpr.linei.NewLine()
			lpr.strl.Pen.X = lpr.WrapData.PenX

			// bg on start of newline - need to get wrap line rune size
			lpr.state = 2
			lpr.strl.RiClone = true
			lpr.strl.Ru = WrapLineRune
			//lpr.strl.Ru = 0
			wrapLineRuneAdv := lpr.wrapLineRuneAdvance(lpr.strl.Ru)
			lpr.strl.Advance = wrapLineRuneAdv + lpr.advance()
			if ok := fn(); !ok {
				return false
			}
			lpr.strl.RiClone = false

			// set pen to draw position
			lpr.strl.Pen.X += lpr.strl.Advance + runeCut - runeAdv

			// reset original rune
			lpr.strl.Ru = origRu
			lpr.strl.Advance = runeAdv

			// original rune
			lpr.state = 3
			if ok := fn(); !ok {
				return false
			}

			lpr.state = 0
		} else {
			if ok := fn(); !ok {
				return false
			}
		}

		// reset wrapindent counters on newline
		if lpr.strl.Ru == '\n' {
			lpr.WrapData.NotStartingSpaces = false
			lpr.WrapData.PenX = 0
		}
		return true
	})
}
func (lpr *WrapLine2Looper) advance() fixed.Int26_6 {
	return lpr.strl.LineHeight() / 2
}
func (lpr *WrapLine2Looper) wrapLineRuneAdvance(ru rune) fixed.Int26_6 {
	origRu := lpr.strl.Ru
	adv := lpr.strl.Advance

	// restore values
	defer func() {
		lpr.strl.Ru = origRu
		lpr.strl.Advance = adv
	}()

	lpr.strl.Ru = ru
	ok := lpr.strl.CalcAdvance()
	if !ok {
		return 0
	}
	return lpr.strl.Advance
}

type WrapLine2Data struct {
	NotStartingSpaces bool // after first non space char
	PenX              fixed.Int26_6
}

type WrapLine2ColorLooper struct {
	EmbedLooper
	wlinel *WrapLine2Looper
	dl     *DrawLooper
	bgl    *BgLooper
}

func NewWrapLine2ColorLooper(wlinel *WrapLine2Looper, dl *DrawLooper, bgl *BgLooper) *WrapLine2ColorLooper {
	return &WrapLine2ColorLooper{wlinel: wlinel, dl: dl, bgl: bgl}
}
func (lpr *WrapLine2ColorLooper) Loop(fn func() bool) {
	var fg color.Color = nil
	var bg color.Color = color.RGBA{222, 222, 222, 255} // between 198,245
	lpr.OuterLooper().Loop(func() bool {
		switch lpr.wlinel.state {
		case 1, 2:
			if bg != nil {
				lpr.bgl.Bg = bg
			}
		case 3:
			if fg != nil {
				lpr.dl.Fg = fg
			}
		}
		return fn()
	})
}
