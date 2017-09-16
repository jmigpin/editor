package loopers

type Looper interface {
	Loop(func() bool)
	InnerLooper() Looper
	SetInnerLooper(Looper)
}

type EmbedLooper struct {
	inner Looper
}

func (el *EmbedLooper) InnerLooper() {
	return el.inner
}
func (el *EmbedLooper) SetInnerLooper(l Looper) {
	el.inner = l
}
