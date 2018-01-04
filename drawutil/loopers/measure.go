package loopers

import (
	"golang.org/x/image/math/fixed"
)

type Measure struct {
	EmbedLooper
	strl *String
	M    fixed.Point26_6
}

func NewMeasure(strl *String) *Measure {
	return &Measure{strl: strl}
}
func (lpr *Measure) Loop(fn func() bool) {
	var m fixed.Point26_6
	lpr.OuterLooper().Loop(func() bool {
		penXAdv := lpr.strl.PenXAdvance()
		if penXAdv > m.X {
			m.X = penXAdv
		}
		return fn()
	})
	m.Y = lpr.strl.LineY1() // always measures at least one line
	lpr.M = m
}
