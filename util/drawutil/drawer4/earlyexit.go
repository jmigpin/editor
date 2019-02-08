package drawer4

import "github.com/jmigpin/editor/util/mathutil"

type EarlyExit struct {
	d *Drawer
}

func (ee *EarlyExit) Init() {
	ee.d.st.earlyExit.maxY = mathutil.Intf1(ee.d.Offset().Y + ee.d.Bounds().Dy())
}

func (ee *EarlyExit) Iter() {
	if ee.d.st.runeR.pen.Y >= ee.d.st.earlyExit.maxY {
		ee.d.iterStop()
		return
	}
	_ = ee.d.iterNext()
}

func (ee *EarlyExit) End() {}
