package drawutil

import (
	"unicode"

	"golang.org/x/image/math/fixed"
)

// Adds newlines and indented wraplines to StringIterator.
type StringLiner struct {
	iter       *StringIterator
	max        fixed.Point26_6
	line       int
	wrapIndent struct {
		startingSpaces bool
		penX           fixed.Int26_6
	}
	isWrapLineRune bool // used to detect if the cursor is on one
	states         struct {
		comment bool
		//str     bool
		//strEnd  bool
	}
}

func NewStringLiner(face *Face, str string, max *fixed.Point26_6) *StringLiner {
	iter := NewStringIterator(face, str)
	liner := &StringLiner{iter: iter, max: *max}
	liner.wrapIndent.startingSpaces = true
	return liner
}
func (liner *StringLiner) Loop(fn func() bool) {
	liner.iter.Loop(func() bool {

		// (comment,string) states are done here to be saved in the stringcache state, otherwise they shouldn't be here

		// comment state
		if !liner.states.comment {
			if liner.iter.ru == '/' {
				next, ok := liner.iter.LookaheadRune(1)
				if ok && next == '/' {
					liner.states.comment = true
				}
			}
			if liner.iter.ru == '#' {
				liner.states.comment = true
			}
		} else {
			if liner.iter.ru == '\n' {
				liner.states.comment = false
			}
		}

		//// string state
		//if !liner.states.str && !liner.states.comment {
		//if liner.iter.ru == '"' {
		//liner.states.str = true
		//liner.states.strEnd = false
		//}
		//} else {
		//if liner.iter.ru == '"' {
		//// end state on next rune
		//liner.states.strEnd = true
		//} else if liner.states.strEnd {
		//liner.states.str = false
		//}
		//}

		// keep track of indentation for wrapped lines
		if liner.wrapIndent.startingSpaces {
			if unicode.IsSpace(liner.iter.ru) {
				liner.wrapIndent.penX = liner.iter.penEnd.X

				// make the runes always visible instead of letting them go undrawn due to being to the right of max x
				d := liner.iter.penEnd.X - liner.iter.pen.X
				if liner.wrapIndent.penX >= liner.max.X-d {
					liner.wrapIndent.penX = liner.max.X - d
				}

			} else {
				liner.wrapIndent.startingSpaces = false
			}
		}

		// wrap line
		if liner.iter.penEnd.X >= liner.max.X {
			liner.newLine()
			liner.iter.pen.X = liner.wrapIndent.penX // indented wrap
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
			liner.wrapIndent.startingSpaces = true
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
