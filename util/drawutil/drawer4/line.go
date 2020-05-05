package drawer4

type Line struct {
	d *Drawer
}

func (l *Line) Init() {}

func (l *Line) Iter() {
	l.d.st.line.lineStart = false
	if l.d.st.runeR.prevRu == '\n' || l.d.st.runeR.ri == l.d.st.runeR.startRi {
		l.d.st.line.lineStart = true
	}

	if !l.d.iterNext() {
		return
	}
	if l.d.st.runeR.ru == '\n' {
		l.newLine()
	}
}

func (l *Line) End() {}

//----------

func (l *Line) newLine() {
	l.newLineKeepAdv()
	l.d.st.runeR.advance = 0
}

func (l *Line) newLineKeepAdv() {
	st := &l.d.st.runeR
	st.pen.X = l.d.iters.runeR.startingPen().X
	st.pen.Y += l.d.lineHeight
	st.prevRu = 0
	st.kern = 0
	// don't reset advance (keeps rune adv calc needed by linewrap)
}
