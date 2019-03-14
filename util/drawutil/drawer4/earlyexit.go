package drawer4

import "github.com/jmigpin/editor/util/mathutil"

type EarlyExit struct {
	d *Drawer
}

func (ee *EarlyExit) Init() {}

func (ee *EarlyExit) Iter() {
	maxY := mathutil.Intf1(ee.d.bounds.Max.Y)

	// extra line iterated (not visible, useful for header1)
	if ee.d.st.earlyExit.extraLine {
		maxY += ee.d.lineHeight
	}

	if ee.d.st.runeR.pen.Y >= maxY {
		ee.d.iterStop()
		return
	}
	if !ee.d.iterNext() {
		return
	}
}

func (ee *EarlyExit) End() {}
