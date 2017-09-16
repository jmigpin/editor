package loopers

import "golang.org/x/image/math/fixed"

type LineLooper struct {
	EmbedLooper
	strl *StringLooper
	MaxY fixed.Int26_6
	Line int
}

func NewLineLooper(strl *StringLooper, maxY fixed.Int26_6) *LineLooper {
	return &LineLooper{strl: strl, MaxY: maxY}
}
func (lpr *LineLooper) Loop(fn func() bool) {
	lpr.OuterLooper().Loop(func() bool {
		pb := lpr.strl.PenBounds()
		if pb.Min.Y >= lpr.MaxY {
			return false
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
	lpr.Line++
	lpr.strl.Pen.X = 0
	lpr.strl.Pen.Y += lpr.strl.LineHeight()
}
