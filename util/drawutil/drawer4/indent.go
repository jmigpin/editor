package drawer4

import (
	"unicode"
)

type Indent struct {
	d *Drawer
}

func (in *Indent) Init() {}

func (in *Indent) Iter() {
	if in.d.iters.runeR.isNormal() {
		// keep track of indentation for wrapped lines
		if in.d.iters.runeR.isNormal() {
			penXAdv := in.d.st.runeR.pen.X + in.d.st.runeR.advance
			if !in.d.st.indent.notStartingSpaces {
				if unicode.IsSpace(in.d.st.runeR.ru) {
					in.d.st.indent.indent = penXAdv
				} else {
					in.d.st.indent.notStartingSpaces = true
				}
			}
		}

		if in.d.iters.lineWrap.wrapping() {
			if !in.indent() {
				return
			}
		}
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

func (in *Indent) indent() bool {
	// set ident
	pen := &in.d.st.runeR.pen
	pen.X = in.d.st.indent.indent
	// left padding
	pad := in.d.iters.runeR.glyphAdvance('\t')
	startX := in.d.iters.runeR.startingPen().X
	if pen.X == 0 {
		pen.X = startX + pad
	} else {
		pen.X += pad
	}
	// margin on the right
	maxX := in.d.iters.runeR.maxX()
	space := in.d.iters.runeR.glyphAdvance(' ')
	margin := space * 5
	if pen.X > maxX-margin {
		pen.X = maxX - margin
	}
	// margin on the left
	if pen.X < startX {
		pen.X = startX
	}

	// keep state
	rr := in.d.st.runeR
	cc := in.d.st.curColors
	// restore state
	defer func() {
		// TODO: adds startoffsetx after the wraplinerune (loses some hability to identify if there is a space present)
		//sox := mathutil.Intf1(in.d.Opt.RuneReader.StartOffsetX)
		//penX := in.d.st.runeR.pen.X + sox // use the indented penX

		penX := in.d.st.runeR.pen.X // use the indented penX
		in.d.st.runeR = rr          // restore
		in.d.st.runeR.pen.X = penX
		in.d.st.curColors = cc
	}()

	assignColor(&in.d.st.curColors.fg, in.d.Opt.LineWrap.Fg)
	assignColor(&in.d.st.curColors.bg, in.d.Opt.LineWrap.Bg)

	s := string(WrapLineRune)
	return in.d.iters.runeR.insertExtraString(s)
}

var WrapLineRune = rune('←') // positioned at the start of wrapped line (left)
