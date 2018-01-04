package loopers

import (
	"unicode"

	"golang.org/x/image/math/fixed"
)

var WrapLineRune = rune(0) // positioned at the start of wrapped line (left)

type WrapLine struct {
	EmbedLooper
	strl  *String
	linei *Line
	MaxX  fixed.Int26_6

	state int // states used by WrapLineColor

	data WrapLine2Data // positional data for keep/restore
}

func MakeWrapLine(strl *String, linei *Line, maxX fixed.Int26_6) WrapLine {
	return WrapLine{strl: strl, linei: linei, MaxX: maxX}
}
func (lpr *WrapLine) Loop(fn func() bool) {
	wlrAdv := lpr.wrapLineRuneAdvance(WrapLineRune)

	// wrap line margin-to-border minimum
	margin := wlrAdv
	adv, ok := lpr.strl.Face.GlyphAdvance(' ')
	if ok {
		margin = wlrAdv + 16*adv
	}

	// TODO: ideally, have the identation be used if the rest of the line fits, otherwise use space available from the start of the line. (Would still have the issue with long line comments not honoring the wrapline indent).

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
			runeAdvPart1 := runeAdv - runeCut
			sepSpace := runeAdv

			origRu := lpr.strl.Ru
			lpr.strl.RiClone = true

			// bg close to the border - current rune size covers the space
			lpr.state = 1
			lpr.strl.Ru = 0
			lpr.strl.Advance = runeAdvPart1
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
			bgAdv := wlrAdv + (sepSpace - runeAdvPart1)
			lpr.strl.Advance = bgAdv
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
			lpr.state = 0
			lpr.strl.RiClone = false
			lpr.strl.Ru = origRu
			lpr.strl.Pen.X = startPenX + bgAdv
			lpr.strl.Advance = runeAdv
			if ok := fn(); !ok {
				return false
			}

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
func (lpr *WrapLine) wrapLineRuneAdvance(ru rune) fixed.Int26_6 {
	if ru == 0 {
		return 0
	}

	// keep original rune and advance
	origRu := lpr.strl.Ru
	adv := lpr.strl.Advance

	// restore at the end
	defer func() {
		lpr.strl.Ru = origRu
		lpr.strl.Advance = adv
	}()

	// calc advance of rune
	lpr.strl.Ru = ru
	ok := lpr.strl.CalcAdvance()
	if !ok {
		return 0
	}
	return lpr.strl.Advance
}

// Implements PosDataKeeper
func (lpr *WrapLine) KeepPosData() interface{} {
	return lpr.data
}

// Implements PosDataKeeper
func (lpr *WrapLine) RestorePosData(data interface{}) {
	lpr.data = data.(WrapLine2Data)
}

// Implements PosDataKeeper
func (lpr *WrapLine) UpdatePosData() {
}

type WrapLine2Data struct {
	NotStartingSpaces bool          // is after first non space char
	PenX              fixed.Int26_6 // indent size, or first rune position after indent
}
