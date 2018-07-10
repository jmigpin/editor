package widget

import "image"

// Used by ScrollArea.
type Scrollable interface {
	Node

	SetScrollable(x, y bool)
	SetScrollableOffset(image.Point)

	ScrollableOffset() image.Point
	ScrollableSize() image.Point
	ScrollablePagingMargin() int
	ScrollableScrollJump() int
}
