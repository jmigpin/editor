package widget

type Cursor int

const (
	NoneCursor Cursor = iota // none means not set
	DefaultCursor
	NSResizeCursor
	WEResizeCursor
	CloseCursor
	MoveCursor
	PointerCursor
	BeamCursor // text cursor
)

//func SetTreeCursor(ctx CursorContext, node Node, p image.Point) {
//	v := setTreeCursor2(ctx, node, p)
//	if !v {
//		ctx.SetCursor(NoneCursor)
//	}
//}
//func setTreeCursor2(ctx CursorContext, node Node, p image.Point) bool {
//	if !p.In(node.Embed().Bounds) {
//		return false
//	}

//	// execute on childs
//	set := false
//	node.Embed().IterateWrappersReverse(func(c Node) bool {
//		v := setTreeCursor2(ctx, c, p)
//		if v {
//			set = true
//			return false // early stop
//		}
//		return true
//	})

//	// execute on node
//	if !set {
//		nc := node.Embed().Cursor
//		if nc != NoneCursor {
//			set = true
//			ctx.SetCursor(nc)
//		}
//	}

//	return set
//}
