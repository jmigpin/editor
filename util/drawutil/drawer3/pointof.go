package drawer3

import (
	"image"
)

type PointOf struct {
	EExt
	index int
	point image.Point // result
}

func PointOf1(index int) PointOf {
	return PointOf{index: index}
}

func (pof *PointOf) Iterate(r *ExtRunner) {
	// don't stop immediately if it is a clone or it could get the wrong pen
	if r.RR.RiClone() {
		r.NextExt()
		return
	}

	if r.RR.Ri >= pof.index {
		r.Stop()
		return
	}
	r.NextExt()
}

func (pof *PointOf) End(r *ExtRunner) {
	// pen is top/left, use what penbounds is using
	pof.point = r.RR.PenBounds().Min.ToPointFloor()
}
