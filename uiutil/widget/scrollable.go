package widget

import "image"

// Ex: TextArea.
type Scrollable interface {
	Node
	SetScroller(Scroller)
	SetScrollableOffset(image.Point)
	ScrollableSize() image.Point
	ScrollablePagingMargin() int
	ScrollableScrollJump() int
}

// Ex: ScrollArea.
type Scroller interface {
	SetScrollerOffset(image.Point)
}
