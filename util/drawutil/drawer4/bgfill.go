package drawer4

import (
	"github.com/jmigpin/editor/util/imageutil"
)

type BgFill struct {
	d *Drawer
}

func (bgf *BgFill) Init() {}

func (bgf *BgFill) Iter() {
	bgf.iter2()
	if !bgf.d.iterNext() {
		return
	}
}
func (bgf *BgFill) iter2() {
	// skip draw
	if bgf.d.st.runeR.ru < 0 {
		if bgf.d.st.runeR.ru == eofRune {
			// allow painting line at eof position
		} else {
			return
		}
	}

	st := &bgf.d.st.curColors
	if st.lineBg != nil {
		r := bgf.d.iters.runeR.penBoundsRect()
		b := bgf.d.bounds
		r.Min.X = b.Min.X
		r.Max.X = b.Max.X
		r = r.Intersect(b)
		imageutil.FillRectangle(bgf.d.st.drawR.img, &r, st.lineBg)
	}
	if st.bg != nil {
		r := bgf.d.iters.runeR.penBoundsRect()
		r = r.Intersect(bgf.d.bounds)
		imageutil.FillRectangle(bgf.d.st.drawR.img, &r, st.bg)
	}
}

func (bgf *BgFill) End() {}
