package drawer4

type Line struct {
	d *Drawer
}

func (l *Line) Init() {}

func (l *Line) Iter() {
	if !l.d.iterNext() {
		return
	}
	if l.d.st.runeR.ru == '\n' {
		l.newLine()
		l.d.st.runeR.advance = 0
	}
}

func (l *Line) End() {}

//----------

func (l *Line) newLine() {
	st := &l.d.st.runeR
	st.pen.X = l.d.iters.runeR.startX()
	st.pen.Y += l.d.lineHeight
	st.prevRu = 0
	st.kern = 0
}
