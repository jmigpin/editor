package loopers

import "golang.org/x/image/math/fixed"

type LineLooper struct {
	EmbedLooper
	strl    *StringLooper
	offsetX fixed.Int26_6
}

func MakeLineLooper(strl *StringLooper, offsetX fixed.Int26_6) LineLooper {
	return LineLooper{strl: strl, offsetX: offsetX}
}
func (lpr *LineLooper) Loop(fn func() bool) {
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
func (lpr *LineLooper) NewLine() {
	lpr.strl.Pen.X = -lpr.offsetX
	lpr.strl.Pen.Y += lpr.strl.LineHeight()
	lpr.strl.PrevRu = 0
	lpr.strl.Advance = 0
}
