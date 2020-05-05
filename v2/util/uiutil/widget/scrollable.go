package widget

import (
	"image"
)

type Scrollable interface {
	SetScrollable(x, y bool)

	ScrollOffset() image.Point
	SetScrollOffset(image.Point)
	ScrollSize() image.Point
	ScrollViewSize() image.Point
	ScrollPageSizeY(up bool) int
	ScrollWheelSizeY(up bool) int
}

// Used by ScrollArea.
type ScrollableNode interface {
	Node
	Scrollable
}
