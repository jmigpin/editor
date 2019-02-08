package drawer4

type LineWrap struct {
	d *Drawer
}

func (lw *LineWrap) Init() {
	lw.d.st.lineWrap.maxX = lw.d.iters.runeR.maxX()
}

func (lw *LineWrap) Iter() {
	stR := &lw.d.st.runeR
	penXAdv := stR.pen.X + stR.advance

	// pen.x>0 forces at least one rune per line
	if penXAdv > lw.d.st.lineWrap.maxX && stR.pen.X > lw.d.iters.runeR.startX() {
		if !lw.fillBgToMaxX() {
			return
		}
		lw.d.iters.line.newLine()

		// let other iterators know it has wrapped
		lw.d.st.lineWrap.wrapped = true
		defer func() { lw.d.st.lineWrap.wrapped = false }()
	}

	_ = lw.d.iterNext()
}

func (lw *LineWrap) End() {}

//----------

func (lw *LineWrap) fillBgToMaxX() bool {
	// keep state
	rr := lw.d.st.runeR
	cc := lw.d.st.curColors
	// restore state
	defer func() {
		lw.d.st.runeR = rr
		lw.d.st.curColors = cc
	}()

	assignColor(&lw.d.st.curColors.bg, lw.d.Opt.LineWrap.Bg)

	lw.d.st.runeR.ru = 0 // don't draw rune

	// current rune advance covers the space to the border

	lw.d.iters.runeR.pushRiExtra()
	defer lw.d.iters.runeR.popRiExtra()
	return lw.d.iterNext()
}
