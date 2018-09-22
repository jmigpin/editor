package drawer3

import "github.com/jmigpin/editor/util/mathutil"

type EarlyExit struct {
	EExt
	maxY mathutil.Intf
}

func (ee *EarlyExit) Start(r *ExtRunner) {
	ee.maxY = mathutil.Intf1(r.D.Offset().Y + r.D.Bounds().Dy())
}

func (ee *EarlyExit) Iterate(r *ExtRunner) {
	if r.RR.Pen.Y >= ee.maxY {
		r.Stop()
		return
	}
	r.NextExt()
}
