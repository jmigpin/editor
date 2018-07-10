package drawer3

import (
	"github.com/jmigpin/editor/util/mathutil"
)

type IndexOf struct {
	EExt
	index int // result
	point mathutil.PointIntf
}

func IndexOf1(p mathutil.PointIntf) IndexOf {
	return IndexOf{point: p}
}

func (ino *IndexOf) Start(r *ExtRunner) {
	ino.index = -1
}

func (ino *IndexOf) Iterate(r *ExtRunner) {
	if r.RR.RiClone() {
		r.NextExt()
		return
	}

	p := &ino.point
	pb := r.RR.PenBounds()

	// before the start or already passed the line
	if p.Y < pb.Min.Y {
		r.Stop()
		return
	}

	// in the line
	if p.Y < pb.Max.Y {
		// keep closest in the line
		ino.index = r.RR.Ri
		// before the first rune of the line or in the rune
		if p.X < pb.Max.X {
			r.Stop()
			return
		}
	}

	r.NextExt()
}

func (ino *IndexOf) End(r *ExtRunner) {
	if ino.index < 0 {
		ino.index = r.RR.Ri // possibly zero or eos
	}
}
