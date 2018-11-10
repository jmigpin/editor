package widget

import "image"

// Used by ScrollArea.
type ScrollableNode interface {
	Node
	Scrollable
}

type Scrollable interface {
	SetScrollable(x, y bool)
	SetScrollableOffset(image.Point)

	ScrollableOffset() image.Point
	ScrollableSize() image.Point
	ScrollablePagingMargin() int
	ScrollableScrollJump() int
}
