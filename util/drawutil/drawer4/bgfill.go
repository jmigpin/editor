package drawer4

import (
	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/mathutil"
)

type BgFill struct {
	d *Drawer
}

func (bgf *BgFill) Init() {}
func (bgf *BgFill) Iter() {
	st := &bgf.d.st.curColors
	if st.bg != nil {
		offset := mathutil.PIntf2(bgf.d.Offset())
		pos := bgf.d.Bounds().Min
		pb := bgf.d.iters.runeR.offsetPenBoundsRect(offset, pos)
		pb = pb.Intersect(bgf.d.Bounds())
		imageutil.FillRectangle(bgf.d.st.drawR.img, &pb, st.bg)
	}
	_ = bgf.d.iterNext()
}
func (bgf *BgFill) End() {}
