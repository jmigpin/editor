package loopers

type LineLooper struct {
	EmbedLooper
	strl *StringLooper
}

func NewLineLooper(strl *StringLooper) *LineLooper {
	return &LineLooper{strl: strl}
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
	lpr.strl.Pen.X = 0
	lpr.strl.Pen.Y += lpr.strl.LineHeight()
	lpr.strl.PrevRu = 0
	lpr.strl.Advance = 0
}
