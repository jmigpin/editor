package uiutil

import (
	"image"

	"github.com/jmigpin/editor/uiutil/widget"
)

func PaintIfNeeded(node widget.Node, painted func(*image.Rectangle)) {
	if node.Marks().NeedsPaint() {
		if widget.PaintTree(node) {
			b := node.Bounds()
			painted(&b)
		}
	} else if node.Marks().ChildNeedsPaint() {
		node.Marks().SetChildNeedsPaint(false)
		for _, child := range node.Childs() {
			PaintIfNeeded(child, painted)
		}
	}
}
