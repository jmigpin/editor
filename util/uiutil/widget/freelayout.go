package widget

type FreeLayout struct {
	ENode
}

func NewFreeLayout() *FreeLayout {
	return &FreeLayout{}
}

func (fl *FreeLayout) Layout() {
	fl.IterateWrappers2(func(child Node) {
		m := child.Measure(fl.Bounds.Size())
		b := fl.Bounds
		b.Max = b.Min.Add(m)
		b = b.Intersect(fl.Bounds)
		child.Embed().Bounds = b
	})
}
