package widget

import "image"

type Scrollable interface {
	SetScrollable(x, y bool)
	SetScrollableOffset(image.Point)

	ScrollableOffset() image.Point
	ScrollableSize() image.Point
	ScrollableViewSize() image.Point
	ScrollablePagingMargin() int // TODO: up arg
	ScrollableScrollJump() int   // TODO: up arg
}

// Used by ScrollArea.
type ScrollableNode interface {
	Node
	Scrollable
}
