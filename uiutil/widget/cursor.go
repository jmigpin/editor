package widget

import (
	"container/list"
)

type Cursor int

const (
	NoCursor Cursor = iota
	DefaultCursor
	NSResizeCursor
	WEResizeCursor
	CloseCursor
	MoveCursor
	PointerCursor
	TextCursor
)

type CursorStack struct {
	list.List
}

func (cs *CursorStack) Push(c Cursor) *CursorRef {
	elem := cs.PushBack(c)
	if elem == nil {
		panic("cursor not inserted")
	}
	return &CursorRef{elem}
}

func (cs *CursorStack) Pop(cr *CursorRef) {
	if cr == nil || cr.elem == nil {
		return
	}
	cs.Remove(cr.elem)
	cr.elem = nil
}

func (cs *CursorStack) SetTop(ctx Context) {
	if cs.Len() == 0 {
		ctx.SetCursor(NoCursor)
	} else {
		ctx.SetCursor(cs.Back().Value.(Cursor))
	}
}

type CursorRef struct {
	elem *list.Element
}

var CursorStk CursorStack
