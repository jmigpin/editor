package drawer4

// Current colors
type CurColors struct {
	d *Drawer
}

func (cc *CurColors) Init() {}

func (cc *CurColors) Iter() {
	st := &cc.d.st.curColors
	st.fg = cc.d.fg
	st.bg = nil
	st.lineBg = nil
	if !cc.d.iterNext() {
		return
	}
}

func (cc *CurColors) End() {}
