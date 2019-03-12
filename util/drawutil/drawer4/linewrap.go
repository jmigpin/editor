package drawer4

type LineWrap struct {
	d *Drawer
}

func (lw *LineWrap) Init() {
	lw.d.st.lineWrap.wrapRi = -1
}

func (lw *LineWrap) Iter() {
	if lw.d.iters.runeR.isNormal() {
		// pen.x>startpen.x forces at least one rune on line start
		stR := &lw.d.st.runeR
		penXAdv := stR.pen.X + stR.advance
		maxX := lw.d.iters.runeR.maxX()
		if penXAdv > maxX && stR.pen.X > lw.d.iters.runeR.startingPen().X {
			// wrapping at ri
			lw.d.st.lineWrap.wrapRi = lw.d.st.runeR.ri

			if !lw.preLineWrap() {
				return
			}

			lw.d.iters.line.newLineKeepAdv()

			if !lw.postLineWrap() {
				return
			}
		}
	}
	if !lw.d.iterNext() {
		return
	}
}

func (lw *LineWrap) End() {}

//----------

// used by indent iterator
func (lw *LineWrap) wrapping() bool {
	return lw.d.st.lineWrap.wrapRi == lw.d.st.runeR.ri
}

func (lw *LineWrap) preLineWrap() bool {
	// fill bg to max x
	// keep state
	rr := lw.d.st.runeR
	cc := lw.d.st.curColors
	// restore state
	defer func() {
		lw.d.st.runeR = rr
		lw.d.st.curColors = cc
	}()
	// color
	assignColor(&lw.d.st.curColors.bg, lw.d.Opt.LineWrap.Bg)
	// draw only the background, use space rune
	lw.d.st.runeR.ru = ' '

	// prelinewrap flag
	lw.d.st.lineWrap.preLineWrap = true
	defer func() { lw.d.st.lineWrap.preLineWrap = false }()

	// current rune advance covers the space to the border
	return lw.d.iterNextExtra()
}

func (lw *LineWrap) postLineWrap() bool {
	// don't draw this extra rune
	rr := lw.d.st.runeR
	lw.d.st.runeR.ru = 0 // don't draw rune
	defer func() { lw.d.st.runeR = rr }()

	// linestart flag
	lw.d.st.lineWrap.postLineWrap = true
	defer func() { lw.d.st.lineWrap.postLineWrap = false }()

	return lw.d.iterNextExtra()
}
