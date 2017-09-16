package loopers

import "golang.org/x/image/math/fixed"

type MeasureLooper struct {
	Looper Looper
	strl   *StringLooper
	max    *fixed.Point26_6
	M      *fixed.Point26_6
}

func NewMeasureLooper(strl *StringLooper, max *fixed.Point26_6) *MeasureLooper {
	return &MeasureLooper{strl: strl, max: max}
}
func (lpr *MeasureLooper) Loop(fn func() bool) {
	var m fixed.Point26_6
	lpr.Looper.Loop(func() bool {
		penXAdv := lpr.strl.PenXAdvance()
		if penXAdv > m.X {
			m.X = penXAdv
		}
		return fn()
	})
	pen := lpr.strl.PenBounds()
	m.Y = pen.Max.Y

	// enforce limits
	if m.X > lpr.max.X {
		m.X = lpr.max.X
	}
	if m.Y > lpr.max.Y {
		m.Y = lpr.max.Y
	}

	lpr.M = &m
}
