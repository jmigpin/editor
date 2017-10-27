package loopers

import (
	"golang.org/x/image/math/fixed"
)

type EarlyExitLooper struct {
	EmbedLooper
	strl *StringLooper
	maxY fixed.Int26_6
}

func (lpr *EarlyExitLooper) Init(strl *StringLooper, maxY fixed.Int26_6) {
	*lpr = EarlyExitLooper{strl: strl, maxY: maxY}
}
func (lpr *EarlyExitLooper) Loop(fn func() bool) {
	lpr.OuterLooper().Loop(func() bool {
		// early exit
		minY := lpr.strl.LineY0()
		if minY >= lpr.maxY {
			return false
		}
		return fn()
	})
}
