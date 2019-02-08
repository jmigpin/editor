package drawer4

// Current colors
type CurColors struct {
	d *Drawer
}

func (cc *CurColors) Init() {}

func (cc *CurColors) Iter() {
	st := &cc.d.st.curColors
	st.fg = st.startFg
	st.bg = nil
	_ = cc.d.iterNext()
}

func (cc *CurColors) End() {}
