package drawer4

type LineWrap struct {
	d *Drawer
}

func (lw *LineWrap) Init() {}

func (lw *LineWrap) Iter() {
	if lw.d.Opt.LineWrap.On && lw.d.iters.runeR.isNormal() {
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

func (lw *LineWrap) preLineWrap() bool {
	// draw only the background, use space rune
	ru := lw.d.st.runeR.ru // keep state
	defer func() { lw.d.st.runeR.ru = ru }()
	lw.d.st.runeR.ru = ' ' // draw only the background, use space rune

	cc := lw.d.st.curColors
	defer func() { lw.d.st.curColors = cc }()
	assignColor(&lw.d.st.curColors.bg, lw.d.Opt.LineWrap.Bg)

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

	s := string(WrapLineRune)
	return lw.d.iters.runeR.insertExtraString(s)
}

var WrapLineRune = rune('←') // positioned at the start of wrapped line (left)
