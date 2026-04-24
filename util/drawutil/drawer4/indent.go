package drawer4

import (
	"unicode"

	"github.com/jmigpin/editor/util/mathutil"
)

var useSpaceForMargin bool

type Indent struct {
	d *Drawer
}

func (in *Indent) Init() {}

func (in *Indent) Iter() {
	if in.d.iters.runeR.isNormal() {
		// keep track of indentation for wrapped lines
		penXAdv := in.d.st.runeR.pen.X + in.d.st.runeR.advance
		if !in.d.st.indent.notStartingSpaces {
			if in.d.st.lineWrap.wrapping {
				in.d.st.indent.notStartingSpaces = true
				in.d.st.indent.indent = 0 // ensure being able to view content
			} else {
				if unicode.IsSpace(in.d.st.runeR.ru) {
					in.d.st.indent.indent = penXAdv
				} else {
					in.d.st.indent.notStartingSpaces = true
				}
			}
		}
	}

	if in.d.st.lineWrap.postLineWrap {
		// Wrapped while still scanning only starting spaces: keep wrapped lines visible by using base indent 0 (+tab padding in indent()).
		if !in.d.st.indent.notStartingSpaces {
			in.d.st.indent.notStartingSpaces = true
			in.d.st.indent.indent = 0
		}

		in.indent()
	}

	if !in.d.iterNext() {
		return
	}

	if in.d.iters.runeR.isNormal() {
		if in.d.st.runeR.ru == '\n' {
			in.d.st.indent.notStartingSpaces = false
			in.d.st.indent.indent = 0
		}
	}
}

func (in *Indent) End() {}

//----------

func (in *Indent) indent() {
	// set ident
	pen := &in.d.st.runeR.pen
	pen.X = in.d.st.indent.indent
	// left padding
	pad := mathutil.Intf(float64(in.d.iters.runeR.glyphAdvance('\t')) * WrapLineIndentTabs)
	startX := in.d.iters.runeR.startingPen().X
	if pen.X == 0 {
		pen.X = startX + pad
	} else {
		pen.X += pad
	}
	// margin on the right
	marginRune := 'W'
	if useSpaceForMargin { // for tests
		marginRune = ' '
	}
	margin := 10 * in.d.iters.runeR.glyphAdvance(marginRune)
	maxX := in.d.iters.runeR.maxX() - margin
	if pen.X > maxX {
		pen.X = maxX
	}
	// margin on the left
	if pen.X < startX {
		pen.X = startX
	}
}
