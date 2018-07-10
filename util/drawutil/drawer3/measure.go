package drawer3

import (
	"image"

	"github.com/jmigpin/editor/util/mathutil"
)

type Measure struct {
	EExt
	measure image.Point // result
	maxPen  mathutil.PointIntf
}

func (m *Measure) Start(r *ExtRunner) {
	m.maxPen = mathutil.PointIntf{}
	m.measure = image.Point{}
}

func (m *Measure) Iterate(r *ExtRunner) {
	penXAdv := r.RR.Pen.X + r.RR.Advance
	if penXAdv > m.maxPen.X {
		m.maxPen.X = penXAdv
	}
	r.NextExt()
}
func (m *Measure) End(r *ExtRunner) {
	// has at least one line height, but x could be zero (penbounds empty)
	m.maxPen.Y = r.RR.Pen.Y + r.RR.LineHeight
	m.measure = m.maxPen.ToPointCeil()
}
