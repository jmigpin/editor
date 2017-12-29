package loopers

import (
	"golang.org/x/image/math/fixed"
)

type EarlyExit struct {
	EmbedLooper
	strl *String
	maxY fixed.Int26_6
}

func MakeEarlyExit(strl *String, maxY fixed.Int26_6) EarlyExit {
	return EarlyExit{strl: strl, maxY: maxY}
}
func (lpr *EarlyExit) Loop(fn func() bool) {
	lpr.OuterLooper().Loop(func() bool {
		// early exit
		minY := lpr.strl.LineY0()
		if minY >= lpr.maxY {
			return false
		}
		return fn()
	})
}
