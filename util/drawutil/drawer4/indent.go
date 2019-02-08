package drawer4

import (
	"unicode"

	"github.com/jmigpin/editor/util/drawutil/drawer3"
	"github.com/jmigpin/editor/util/mathutil"
)

type Indent struct {
	d *Drawer
}

func (in *Indent) Init() {
	in.reset()
}

func (in *Indent) Iter() {
	penXAdv := in.d.st.runeR.pen.X + in.d.st.runeR.advance

	// keep track of indentation for wrapped lines
	if !in.d.st.indent.notStartingSpaces {
		if unicode.IsSpace(in.d.st.runeR.ru) {
			in.d.st.indent.indent = penXAdv
		} else {
			in.d.st.indent.notStartingSpaces = true
		}
	}

	// line was wrapped
	if in.d.st.lineWrap.wrapped {
		if !in.indent() {
			return
		}
	}

	if !in.d.iterNext() {
		return
	}

	// reset tracking starting spaces on newline
	if in.d.st.runeR.ru == '\n' {
		in.reset()
	}
}

func (in *Indent) End() {}

//----------

func (in *Indent) reset() {
	in.d.st.indent.notStartingSpaces = false
	in.d.st.indent.indent = 0
}

func (in *Indent) indent() bool {
	// set ident
	{
		pen := &in.d.st.runeR.pen
		startX := mathutil.Intf1(0) // no startx, joined with border
		if drawer3.WrapLineRune == 0 {
			startX = pen.X // uses startx
		}
		pen.X = in.d.st.indent.indent
		maxX := in.d.iters.runeR.maxX()
		space := in.d.iters.runeR.glyphAdvance(' ') * 5
		if pen.X > maxX-space {
			pen.X = maxX - space
		}
		if pen.X < startX {
			pen.X = startX
		}
	}

	// keep state
	rr := in.d.st.runeR
	cc := in.d.st.curColors
	// restore state
	defer func() {
		penX := in.d.st.runeR.pen.X
		in.d.st.runeR = rr
		in.d.st.runeR.pen.X = penX
		in.d.st.curColors = cc
	}()

	assignColor(&in.d.st.curColors.fg, in.d.Opt.LineWrap.Fg)
	assignColor(&in.d.st.curColors.bg, in.d.Opt.LineWrap.Bg)

	s := string(drawer3.WrapLineRune)
	return in.d.iters.runeR.insertExtraString(s)
}
