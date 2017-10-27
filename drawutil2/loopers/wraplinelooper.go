package loopers

import (
	"image/color"
	"unicode"

	"golang.org/x/image/math/fixed"
)

var WrapLineRune = rune(0) // positioned at the start of wrapped line (left)

type WrapLineLooper struct {
	EmbedLooper
	strl  *StringLooper
	linei *LineLooper
	MaxX  fixed.Int26_6

	state int

	data WrapLine2Data // positional data for keep/restore
}

func (lpr *WrapLineLooper) Init(strl *StringLooper, linei *LineLooper, maxX fixed.Int26_6) {
	*lpr = WrapLineLooper{strl: strl, linei: linei, MaxX: maxX}
}
func (lpr *WrapLineLooper) Loop(fn func() bool) {
	wlrAdv := lpr.wrapLineRuneAdvance(WrapLineRune)

	// wrap line margin-to-border minimum
	margin := wlrAdv
	adv, ok := lpr.strl.Face.GlyphAdvance(' ')
	if ok {
		margin = wlrAdv + 8*adv
	}

	lpr.OuterLooper().Loop(func() bool {
		lpr.state = 0

		penXAdv := lpr.strl.PenXAdvance()

		// keep track of indentation for wrapped lines
		if !lpr.data.NotStartingSpaces {
			if unicode.IsSpace(lpr.strl.Ru) {
				lpr.data.PenX = penXAdv
			} else {
				lpr.data.NotStartingSpaces = true
			}
		}

		// wrap line
		if penXAdv > lpr.MaxX && lpr.strl.Ri > 0 {
			runeAdv := penXAdv - lpr.strl.Pen.X
			runeCut := penXAdv - lpr.MaxX

			origRu := lpr.strl.Ru
			lpr.strl.RiClone = true

			// bg close to the border - current rune size covers the space
			lpr.state = 1
			lpr.strl.Ru = 0
			lpr.strl.Advance = runeAdv - runeCut // accurate advance for measure
			if ok := fn(); !ok {
				return false
			}

			// newline
			lpr.linei.NewLine()
			lpr.strl.Pen.X = lpr.data.PenX

			// make wrap line rune always visible
			if lpr.strl.Pen.X >= lpr.MaxX-margin {
				lpr.strl.Pen.X = lpr.MaxX - margin
				if lpr.strl.Pen.X < 0 {
					lpr.strl.Pen.X = 0
				}
			}

			startPenX := lpr.strl.Pen.X

			// bg on start of newline
			lpr.state = 2
			lpr.strl.Ru = 0
			lpr.strl.Pen.X = startPenX
			fixedAdv := wlrAdv + lpr.advance()
			movingAdv := fixedAdv + runeCut - runeAdv
			if movingAdv < wlrAdv {
				movingAdv = wlrAdv
			}
			lpr.strl.Advance = movingAdv // moving bg
			//lpr.strl.Advance = fixedAdv // fixed bg (debug)
			if ok := fn(); !ok {
				return false
			}

			// wraplinerune
			lpr.state = 3
			lpr.strl.Ru = WrapLineRune
			lpr.strl.Pen.X = startPenX
			lpr.strl.Advance = wlrAdv
			if ok := fn(); !ok {
				return false
			}

			// original rune
			lpr.state = 4
			lpr.strl.RiClone = false
			lpr.strl.Ru = origRu
			lpr.strl.Pen.X = startPenX + movingAdv
			lpr.strl.Advance = runeAdv
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
			lpr.data.NotStartingSpaces = false
			lpr.data.PenX = 0
		}
		return true
	})
}
func (lpr *WrapLineLooper) advance() fixed.Int26_6 {
	return lpr.strl.LineHeight() / 2
}
func (lpr *WrapLineLooper) wrapLineRuneAdvance(ru rune) fixed.Int26_6 {
	if ru == 0 {
		return lpr.strl.LineHeight() / 2
	}
	origRu := lpr.strl.Ru
	adv := lpr.strl.Advance
	defer func() {
		// restore values
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

// Implements PosDataKeeper
func (lpr *WrapLineLooper) KeepPosData() interface{} {
	return lpr.data
}

// Implements PosDataKeeper
func (lpr *WrapLineLooper) RestorePosData(data interface{}) {
	lpr.data = data.(WrapLine2Data)
}

// Implements PosDataKeeper
func (lpr *WrapLineLooper) UpdatePosData() {
}

type WrapLine2Data struct {
	NotStartingSpaces bool          // is after first non space char
	PenX              fixed.Int26_6 // indent size, or first rune position after indent
}

type WrapLineColorLooper struct {
	EmbedLooper
	wlinel *WrapLineLooper
	dl     *DrawLooper
	bgl    *BgLooper
	Fg, Bg color.Color
}

func (lpr *WrapLineColorLooper) Init(wlinel *WrapLineLooper, dl *DrawLooper, bgl *BgLooper) {
	*lpr = WrapLineColorLooper{wlinel: wlinel, dl: dl, bgl: bgl}
}
func (lpr *WrapLineColorLooper) Loop(fn func() bool) {
	lpr.OuterLooper().Loop(func() bool {
		switch lpr.wlinel.state {
		case 1, 2:
			if lpr.Bg != nil {
				lpr.bgl.Bg = lpr.Bg
			}
		case 3, 4:
			if lpr.Fg != nil {
				lpr.dl.Fg = lpr.Fg
			}
		}
		return fn()
	})
}
