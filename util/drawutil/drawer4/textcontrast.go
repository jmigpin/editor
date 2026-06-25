package drawer4

import "github.com/jmigpin/editor/util/imageutil"

type TextContrast struct {
	d *Drawer
}

func (tc *TextContrast) Init() {}

func (tc *TextContrast) Iter() {
	tc.adjust()
	if !tc.d.iterNext() {
		return
	}
}

func (tc *TextContrast) End() {}

func (tc *TextContrast) adjust() {
	if !tc.d.Opt.TextContrast.On {
		return
	}
	st := &tc.d.st.curColors
	bg := st.bg
	if bg == nil {
		bg = st.lineBg
	}
	if bg == nil {
		bg = tc.d.Opt.TextContrast.Bg
	}
	if bg == nil {
		return
	}
	st.fg = imageutil.EnsureContrastColor(st.fg, bg)
}
