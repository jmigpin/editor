package widget

import (
	"image"
)

type FloatBox struct {
	ShellEmbedNode
	AlignRight bool

	ref Node
}

func (fb *FloatBox) Init(ref, child Node) {
	*fb = FloatBox{ref: ref}
	fb.SetWrapper(fb)
	fb.Append(child)
}
func (fb *FloatBox) CalcChildsBounds() {
	rb := fb.ref.Bounds()
	fbb := fb.Bounds()

	// bounds bellow reference node
	b := image.Rect(rb.Min.X, rb.Max.Y, rb.Min.X+fbb.Dx(), rb.Max.Y+fbb.Dy())

	// measure child
	child := fb.FirstChild()
	m := child.Measure(b.Size()).Add(b.Min)
	b2 := image.Rect(b.Min.X, b.Min.Y, m.X, m.Y)
	if b2.Max.X > fbb.Max.X {
		diff := image.Point{b2.Max.X - fbb.Max.X, 0}
		b2 = b2.Sub(diff)
	}
	b2 = b2.Intersect(fbb)
	child.SetBounds(&b2)

	// keep same bounds as the child
	fb.SetBounds(&b2)

	child.CalcChildsBounds()
}
