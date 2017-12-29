package loopers

import "golang.org/x/image/math/fixed"

type Line struct {
	EmbedLooper
	strl    *String
	offsetX fixed.Int26_6
}

func MakeLine(strl *String, offsetX fixed.Int26_6) Line {
	return Line{strl: strl, offsetX: offsetX}
}
func (lpr *Line) Loop(fn func() bool) {
	lpr.OuterLooper().Loop(func() bool {
		if lpr.strl.RiClone {
			return fn()
		}
		if ok := fn(); !ok {
			return false
		}
		if lpr.strl.Ru == '\n' {
			lpr.NewLine()
		}
		return true
	})
}
func (lpr *Line) NewLine() {
	lpr.strl.Pen.X = -lpr.offsetX
	lpr.strl.Pen.Y += lpr.strl.LineHeight()
	lpr.strl.PrevRu = 0
	lpr.strl.Advance = 0
}
