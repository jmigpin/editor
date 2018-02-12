package loopers

import (
	"image/color"
	"unicode"

	"golang.org/x/image/math/fixed"
)

var WrapLineRune = rune(0) // positioned at the start of wrapped line (left)

type WrapLine struct {
	EmbedLooper
	strl  *String
	linel *Line
	MaxX  fixed.Int26_6

	data WrapLineData // positional data for keep/restore
}

func MakeWrapLine(strl *String, linei *Line, maxX fixed.Int26_6) WrapLine {
	return WrapLine{strl: strl, linel: linei, MaxX: maxX}
}
func (lpr *WrapLine) Loop(fn func() bool) {
	lpr.OuterLooper().Loop(func() bool {
		// don't act if other clones are active (ex: annotations)
		if lpr.strl.IsRiClone() {
			return fn()
		}

		lpr.data.state = WLStateNormal

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
			// wrap line rune advance
			wlrAdv, _ := lpr.strl.Face.GlyphAdvance(WrapLineRune)

			// wrap line margin-to-border minimum
			margin := wlrAdv
			wAdv, _ := lpr.strl.Face.GlyphAdvance('W')
			margin = wlrAdv + 8*wAdv

			// helper vars
			runeAdv := penXAdv - lpr.strl.Pen.X
			runeAdv1 := lpr.MaxX - lpr.strl.Pen.X
			//runeAdv2 := penXAdv - lpr.MaxX
			//sepSpace := runeAdv // dynamic width leads to back an forth adjustments (annoying)
			//sepSpace, _ := lpr.strl.Face.GlyphAdvance('W') // fixed width of a wide rune

			//// special case for wider rune tab
			//sepSpace2, _ := lpr.strl.Face.GlyphAdvance(lpr.strl.Ru)
			//if sepSpace2 > sepSpace {
			//	sepSpace = sepSpace2
			//}

			origRu := lpr.strl.Ru
			lpr.strl.PushRiClone()

			// bg close to the border - current rune size covers the space
			lpr.data.state = WLStateLine1Bg
			lpr.strl.Ru = 0
			lpr.strl.Advance = runeAdv1
			if ok := fn(); !ok {
				return false
			}

			// newline
			lpr.linel.NewLine()
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
			lpr.data.state = WLStateLine2Bg
			lpr.strl.Ru = 0
			lpr.strl.Pen.X = startPenX
			//bgAdv := wlrAdv + (sepSpace - runeAdv1) // dynamic size
			bgAdv := wlrAdv // fixed size
			lpr.strl.Advance = bgAdv
			if ok := fn(); !ok {
				return false
			}

			// wraplinerune
			lpr.data.state = WLStateLine2Rune
			lpr.strl.Ru = WrapLineRune
			lpr.strl.Pen.X = startPenX
			lpr.strl.Advance = wlrAdv
			if ok := fn(); !ok {
				return false
			}

			// original rune
			lpr.data.state = WLStateNormal
			lpr.strl.PopRiClone()
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

// Implements PosDataKeeper
func (lpr *WrapLine) KeepPosData() interface{} {
	return lpr.data
}

// Implements PosDataKeeper
func (lpr *WrapLine) RestorePosData(data interface{}) {
	lpr.data = data.(WrapLineData)
}

type WLState int

const (
	WLStateNormal WLState = iota
	WLStateLine1Bg
	WLStateLine2Bg
	WLStateLine2Rune
)

type WrapLineData struct {
	state             WLState
	NotStartingSpaces bool          // is after first non space char
	PenX              fixed.Int26_6 // indent size, or first rune position after indent
}

type WrapLineColor struct {
	EmbedLooper
	wlinel *WrapLine
	dl     *Draw
	bgl    *Bg
	opt    *WrapLineOpt
}

func MakeWrapLineColor(wlinel *WrapLine, dl *Draw, bgl *Bg, opt *WrapLineOpt) WrapLineColor {
	return WrapLineColor{wlinel: wlinel, dl: dl, bgl: bgl, opt: opt}
}
func (lpr *WrapLineColor) Loop(fn func() bool) {
	lpr.OuterLooper().Loop(func() bool {
		switch lpr.wlinel.data.state {
		case WLStateLine1Bg, WLStateLine2Bg, WLStateLine2Rune:
			if lpr.opt.Fg != nil {
				lpr.dl.Fg = lpr.opt.Fg
			}
			if lpr.opt.Bg != nil {
				lpr.bgl.Bg = lpr.opt.Bg
			}
		}
		return fn()
	})
}

type WrapLineOpt struct {
	Fg, Bg color.Color
}
