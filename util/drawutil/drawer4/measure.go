package drawer4

type Measure struct {
	d *Drawer
}

func (m *Measure) Init() {
	pb := m.d.iters.runeR.penBounds()
	m.d.st.measure.penMax = pb.Max
}

func (m *Measure) Iter() {
	if m.d.iters.runeR.isNormal() {
		penXAdv := m.d.st.runeR.pen.X + m.d.st.runeR.advance
		if penXAdv > m.d.st.measure.penMax.X {
			m.d.st.measure.penMax.X = penXAdv
		}
	}
	if !m.d.iterNext() {
		return
	}
}

func (m *Measure) End() {
	pb := m.d.iters.runeR.penBounds()
	m.d.st.measure.penMax.Y = pb.Max.Y
}
