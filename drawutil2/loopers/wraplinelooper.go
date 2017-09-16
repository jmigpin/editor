package loopers

import (
	"unicode"

	"golang.org/x/image/math/fixed"
)

var WrapLineRune = rune(0x21b3) // positioned at the left, points to the right
const GoFontWrapLineRune = rune(8594)

type WrapLineLooper struct {
	EmbedLooper
	strl           *StringLooper
	linei          *LineLooper
	MaxX           fixed.Int26_6
	WrapIndent     WrapIndent
	IsWrapLineRune bool
}

func NewWrapLineLooper(strl *StringLooper, linei *LineLooper, maxX fixed.Int26_6) *WrapLineLooper {
	return &WrapLineLooper{strl: strl, linei: linei, MaxX: maxX}
}
func (lpr *WrapLineLooper) Loop(fn func() bool) {
	// wrap line margin constant
	margin := fixed.I(30)
	adv, ok := lpr.strl.Face.GlyphAdvance(WrapLineRune)
	if ok {
		margin = adv
	}

	lpr.OuterLooper().Loop(func() bool {
		penXAdv := lpr.strl.PenXAdvance()

		// keep track of indentation for wrapped lines
		if !lpr.WrapIndent.NotStartingSpaces {
			if unicode.IsSpace(lpr.strl.Ru) {
				lpr.WrapIndent.PenX = penXAdv
			} else {
				lpr.WrapIndent.NotStartingSpaces = true
			}
		}

		// wrap line
		if lpr.strl.Ri > 0 && penXAdv > lpr.MaxX {
			lpr.linei.NewLine()
			lpr.strl.Pen.X = lpr.WrapIndent.PenX

			// make wrap line rune always visible
			if lpr.strl.Pen.X >= lpr.MaxX-margin {
				lpr.strl.Pen.X = lpr.MaxX - margin
				if lpr.strl.Pen.X < 0 {
					lpr.strl.Pen.X = 0
				}
			}

			// keep original rune
			origRu := lpr.strl.Ru

			// insert wrap line symbol at beginning of the line
			lpr.strl.RiClone = true
			lpr.strl.Ru = WrapLineRune
			lpr.IsWrapLineRune = true
			lpr.strl.PrevRu = rune(0)
			if ok := lpr.strl.Iterate(fn); !ok {
				return false
			}
			lpr.IsWrapLineRune = false
			lpr.strl.RiClone = false

			// continue with original rune - no newline
			lpr.strl.Ru = origRu
			lpr.strl.AddKern()
			lpr.strl.CalcAdvance()
			// penXAdv = lpr.PenXAdvance() // not used below
		}

		if ok := fn(); !ok {
			return false
		}

		// reset wrapindent counters on newline
		if lpr.strl.Ru == '\n' {
			lpr.WrapIndent.NotStartingSpaces = false
			lpr.WrapIndent.PenX = 0
		}
		return true
	})
}

type WrapIndent struct {
	NotStartingSpaces bool // after first non space char
	PenX              fixed.Int26_6
}
