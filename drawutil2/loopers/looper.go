package loopers

type Looper interface {
	Loop(func() bool)
	OuterLooper() Looper
	SetOuterLooper(Looper)
}

type EmbedLooper struct {
	outer Looper
}

func (el *EmbedLooper) OuterLooper() Looper {
	return el.outer
}
func (el *EmbedLooper) SetOuterLooper(l Looper) {
	el.outer = l
}
func (el *EmbedLooper) Loop(fn func() bool) {
	for {
		if !fn() {
			break
		}
	}
}
