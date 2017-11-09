package uiutil

import (
	"image"

	"github.com/jmigpin/editor/uiutil/widget"
)

func PaintIfNeeded(node widget.Node, painted func(*image.Rectangle)) {
	if node.Embed().NeedsPaint() {
		if widget.PaintTree(node) {
			b := node.Bounds()
			painted(&b)
		}
	} else if node.Embed().ChildNeedsPaint() {
		node.Embed().UnmarkChildNeedsPaint()
		for _, child := range node.Childs() {
			PaintIfNeeded(child, painted)
		}
	}
}
