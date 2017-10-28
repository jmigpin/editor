package widget

import (
	"image"
)

// First child is bottom layer.
type MultiLayer struct {
	ContainerEmbedNode
}

func (ml *MultiLayer) Measure(hint image.Point) image.Point {
	panic("calling measure on multilayer")
}
func (ml *MultiLayer) CalcChildsBounds() {
	// all childs get full bounds
	u := ml.Bounds()
	for _, n := range ml.Childs() {
		n.SetBounds(&u)
		n.CalcChildsBounds()
	}
}
