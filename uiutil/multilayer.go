package uiutil

import (
	"image"

	"github.com/jmigpin/editor/uiutil/widget"
)

type MultiLayer struct {
	Layers []widget.Node
}

func (ml *MultiLayer) CalcChildsBounds() {
	// first layer last
	for i := len(ml.Layers) - 1; i >= 0; i-- {
		l := ml.Layers[i]
		_ = l
	}
}
func (ml *MultiLayer) PaintIfNeeded() {
	// first layer first
	for _, l := range ml.Layers {
		PaintIfNeeded(l, func(r *image.Rectangle) {
			//painted = true
			//ui.incompleteDraws++
			//ui.win.PutImage(r)
		})
	}
}
func (ml *MultiLayer) ApplyInputEvent(ev interface{}, p image.Point) {
	// first layer last
	for i := len(ml.Layers) - 1; i >= 0; i-- {
		l := ml.Layers[i]
		ApplyInputEventInBounds(l, ev, p)
		// TODO: break if in upper layer, don't go to next layer
	}
}
