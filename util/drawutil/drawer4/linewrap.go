package drawer4

import (
	"unicode"

	"github.com/jmigpin/editor/util/drawutil"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/mathutil"
)

type LineWrap struct {
	d *Drawer
}

func (lw *LineWrap) Init() {}

func (lw *LineWrap) Iter() {
	if lw.d.Opt.LineWrap.On && lw.d.iters.runeR.isNormal() {
		// Prefer word-boundary wrap if enabled
		if lw.shouldWordWrap() {
			lw.d.st.lineWrap.wrapping = true

			if !lw.preLineWrap() {
				return
			}

			lw.d.iters.line.newLineKeepAdv()

			if !lw.postLineWrap() {
				return
			}
		} else {
			// Fallback to existing rune-level wrap
			// pen.x>startpen.x forces at least one rune on line start
			stR := &lw.d.st.runeR
			penXAdv := stR.pen.X + stR.advance
			maxX := lw.d.iters.runeR.maxX()
			if penXAdv > maxX && stR.pen.X > lw.d.iters.runeR.startingPen().X {
				lw.d.st.lineWrap.wrapping = true

				if !lw.preLineWrap() {
					return
				}

				lw.d.iters.line.newLineKeepAdv()

				if !lw.postLineWrap() {
					return
				}
			}
		}
	}

	// after postlinewrap to allow indent to be added
	if lw.d.st.lineWrap.wrapping {
		if !lw.insertWrapRune() {
			return
		}
		// recalc (tab) advance after possible insertions
		lw.d.st.runeR.advance = lw.d.iters.runeR.tabbedGlyphAdvance(lw.d.st.runeR.ru)
	}
	lw.d.st.lineWrap.wrapping = false

	if !lw.d.iterNext() {
		return
	}
}

func (lw *LineWrap) End() {}

//----------

func (lw *LineWrap) breakFn() func(rune) bool {
	return func(r rune) bool {
		return !unicode.IsDigit(r) && !unicode.IsLetter(r)
	}
}

func (lw *LineWrap) shouldWordWrap() bool {
	limit := drawutil.WrapWordLimit
	if limit == 0 {
		return false
	}

	// If the word starts at the start of the line, do rune-level wrap
	if lw.d.st.line.lineStart {
		return false
	}

	st := &lw.d.st.runeR
	bf := lw.breakFn()

	// Only consider if we're at the start of a word (current is not break,  previous is break/start)
	if bf(st.ru) {
		return false
	}
	prevIsBreak := bf(st.prevRu) || isNlOrTermWrap(st.prevRu) || st.prevRu == 0
	if !prevIsBreak {
		return false
	}

	// Look ahead to measure the next word width without consuming input
	rr := &lw.d.iters.runeR
	maxX := rr.maxX()
	x := st.pen.X
	prev := st.prevRu
	i := st.ri

	count := 0
	for {
		ru, sz, err := iorw.ReadRuneAt(lw.d.reader, i)
		if err != nil || sz == 0 {
			break
		}
		if bf(ru) || isNlOrTermWrap(ru) {
			break
		}

		count++
		if count > limit { // too long; don't word-wrap this, let rune-level wrap handle it
			return false
		}

		// apply kern and advance
		k := lw.d.st.runeR.fface.Face.Kern(prev, ru)
		x += mathutil.Intf2(k)
		x += rr.tabbedGlyphAdvance(ru)
		prev = ru
		i += sz

		if x > maxX {
			break
		}
	}

	// Wrap before the word if it doesn't fit
	return x > maxX
}

//----------

func (lw *LineWrap) preLineWrap() bool {
	// draw only the background, use space rune
	ru := lw.d.st.runeR.ru // keep state
	defer func() { lw.d.st.runeR.ru = ru }()
	lw.d.st.runeR.ru = ' ' // draw only the background, use space rune

	cc := lw.d.st.curColors
	defer func() { lw.d.st.curColors = cc }()
	assignColor(&lw.d.st.curColors.bg, lw.d.Opt.LineWrap.Bg)

	// Expand advance to fill up to the right border during pre-wrap
	adv := lw.d.st.runeR.advance
	defer func() { lw.d.st.runeR.advance = adv }()
	maxX := lw.d.iters.runeR.maxX()
	rem := max(0, maxX-lw.d.st.runeR.pen.X)
	lw.d.st.runeR.advance = rem

	lw.d.st.lineWrap.preLineWrap = true
	defer func() { lw.d.st.lineWrap.preLineWrap = false }()

	// current rune advance covers the space to the border
	return lw.d.iterNextExtra()
}

func (lw *LineWrap) postLineWrap() bool {
	// allow post line detection (this rune is not to be drawn)
	ru := lw.d.st.runeR.ru // keep state
	defer func() { lw.d.st.runeR.ru = ru }()
	lw.d.st.runeR.ru = noDrawRune

	lw.d.st.lineWrap.postLineWrap = true
	defer func() { lw.d.st.lineWrap.postLineWrap = false }()

	return lw.d.iterNextExtra()
}

func (lw *LineWrap) insertWrapRune() bool {
	if drawutil.WrapLineRune == 0 {
		return true
	}

	// keep state
	rr := lw.d.st.runeR
	defer func() {
		penX := lw.d.st.runeR.pen.X
		lw.d.st.runeR = rr
		lw.d.st.runeR.pen.X = penX // use the new penX
	}()

	cc := lw.d.st.curColors // keep state
	defer func() { lw.d.st.curColors = cc }()
	assignColor(&lw.d.st.curColors.fg, lw.d.Opt.LineWrap.Fg)
	assignColor(&lw.d.st.curColors.bg, lw.d.Opt.LineWrap.Bg)

	s := string(drawutil.WrapLineRune)
	return lw.d.iters.runeR.insertExtraString(s)
}
