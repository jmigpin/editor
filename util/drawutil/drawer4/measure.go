package drawer4

type Measure struct {
	d *Drawer
}

func (m *Measure) Init() {}

func (m *Measure) Iter() {
	if !m.d.iters.runeR.isRiExtra() {
		penXAdv := m.d.st.runeR.pen.X + m.d.st.runeR.advance
		if penXAdv > m.d.st.measure.penMax.X {
			m.d.st.measure.penMax.X = penXAdv
		}
	}
	_ = m.d.iterNext()
}

func (m *Measure) End() {
	// has at least one line height, but x could be zero (penbounds empty)
	m.d.st.measure.penMax.Y = m.d.st.runeR.pen.Y + m.d.lineHeight
}
