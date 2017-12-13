package widget

import (
	"image"
)

type Cursor int

const (
	NoneCursor Cursor = iota
	NSResizeCursor
	WEResizeCursor
	CloseCursor
	MoveCursor
	PointerCursor
	TextCursor
)

func SetTreeCursor(ctx Context, node Node, p image.Point) {
	v := setTreeCursor2(ctx, node, p)
	if !v {
		ctx.SetCursor(NoneCursor)
	}
}
func setTreeCursor2(ctx Context, node Node, p image.Point) bool {
	if !p.In(node.Embed().Bounds) {
		return false
	}

	// execute on childs
	set := false
	node.Embed().IterChildsReverseStop(func(c Node) bool {
		v := setTreeCursor2(ctx, c, p)
		if v {
			set = true
			return false // early stop
		}
		return true
	})

	// execute on node
	if !set {
		nc := node.Embed().Cursor
		if nc != NoneCursor {
			set = true
			ctx.SetCursor(nc)
		}
	}

	return set
}
