package drawer4

type IndexOf struct {
	d *Drawer
}

func (io *IndexOf) Init() {
	io.d.st.indexOf.index = -1
}

func (io *IndexOf) Iter() {
	if io.d.iters.runeR.isNormal() {
		io.iter2()
	}
	if !io.d.iterNext() {
		return
	}
}

func (io *IndexOf) iter2() {
	p := &io.d.st.indexOf.p
	pb := io.d.iters.runeR.penBounds()
	// before the start
	if p.Y < pb.Min.Y {
		io.d.iterStop()
		return
	}
	// in the line
	if p.Y < pb.Max.Y {
		// keep closest in the line
		io.d.st.indexOf.index = io.d.st.runeR.ri
		// before the first rune of the line or in the rune
		if p.X < pb.Max.X {
			io.d.iterStop()
			return
		}
	}
}

func (io *IndexOf) End() {
	if io.d.st.indexOf.index < 0 {
		io.d.st.indexOf.index = io.d.st.runeR.ri // possibly zero
	}
}
