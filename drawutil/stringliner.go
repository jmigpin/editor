package drawutil

import (
	"unicode"

	"golang.org/x/image/math/fixed"
)

//const wrapLineRightRune = rune(0x21b2) // points to the left
const wrapLineLeftRune = rune(0x21b3) // points to the right

// Adds newlines and indented wraplines to StringIterator.
type StringLiner struct {
	iter           *StringIterator
	max            fixed.Point26_6
	line           int
	wrapIndent     StringLinerWrapIndent
	isWrapLineRune bool // used to detect if the cursor is in one
	states         StringLinerStates
}

type StringLinerWrapIndent struct {
	notStartingSpaces bool // after first non space char
	penX              fixed.Int26_6
}

type StringLinerStates struct {
	// if ever used, needed for comments/strings states
}

func NewStringLiner(face *Face, str string, max *fixed.Point26_6) *StringLiner {
	iter := NewStringIterator(face, str)
	liner := &StringLiner{iter: iter, max: *max}
	return liner
}
func (liner *StringLiner) Loop(fn func() bool) {
	// wrap line margin constant
	wlMargin := fixed.I(30)
	adv, ok := liner.iter.face.GlyphAdvance(' ')
	if ok {
		wlMargin = adv * 7
	}

	liner.iter.Loop(func() bool {
		// keep track of indentation for wrapped lines
		if !liner.wrapIndent.notStartingSpaces {
			if unicode.IsSpace(liner.iter.ru) {
				liner.wrapIndent.penX = liner.iter.penEnd.X
			} else {
				liner.wrapIndent.notStartingSpaces = true
			}
		}

		// wrap line
		if liner.iter.ri > 0 && liner.iter.penEnd.X > liner.max.X {
			liner.newLine()
			liner.iter.pen.X = liner.wrapIndent.penX

			// make runes visible if wrap is beyond max
			if liner.iter.pen.X >= liner.max.X-wlMargin {
				liner.iter.pen.X = liner.max.X - wlMargin
				if liner.iter.pen.X < 0 {
					liner.iter.pen.X = 0
				}
			}

			liner.iter.calcPenEnd()

			// insert wrap line symbol at beginning of the line
			origRu := liner.iter.ru
			liner.iter.ru = wrapLineLeftRune
			liner.isWrapLineRune = true
			liner.iter.calcPenEnd()
			if ok := fn(); !ok {
				return false
			}
			liner.isWrapLineRune = false
			// continue with original rune
			liner.iter.prevRu = liner.iter.ru
			liner.iter.pen = liner.iter.penEnd
			liner.iter.ru = origRu
			liner.iter.addKernToPen()
			liner.iter.calcPenEnd()
		}

		// y bound
		if LineY0(liner.iter.pen.Y, liner.iter.fm) >= liner.max.Y {
			return false
		}

		if ok := fn(); !ok {
			return false
		}

		// new line
		if liner.iter.ru == '\n' {
			liner.newLine()
			liner.wrapIndent.notStartingSpaces = false
			liner.wrapIndent.penX = 0
		}

		return true
	})
}
func (liner *StringLiner) newLine() {
	liner.line++
	liner.iter.pen.X = 0
	liner.iter.pen.Y += LineHeight(liner.iter.fm)
	liner.iter.penEnd = liner.iter.pen
}
