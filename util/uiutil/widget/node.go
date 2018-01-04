package widget

import (
	"container/list"
	"fmt"
	"image"
	"reflect"
	"sync"
)

type Node interface {
	Embed() *EmbedNode

	InsertBefore(n, mark Node)
	Append(n ...Node)
	Remove(Node)
	Swap(Node)

	// Node is an immediate child of this node, while the rectangle is the bounds of the original subchild node that requested the paint.
	OnMarkChildNeedsPaint(Node, *image.Rectangle)

	Measure(hint image.Point) image.Point
	CalcChildsBounds()
	Paint()
	PaintChilds()

	OnInputEvent(ev interface{}, p image.Point) bool
}

type EmbedNode struct {
	Cursor Cursor
	Bounds image.Rectangle

	// Layout testing
	//Pos         image.Point
	//Measurement image.Point

	wrapper Node
	childs  list.List
	elem    *list.Element
	parent  *EmbedNode

	// TODO: pass these to the implementing node
	expand struct{ x, y bool }
	fill   struct{ x, y bool }

	marks     Marks
	marksLock sync.RWMutex
}

func (en *EmbedNode) Embed() *EmbedNode {
	return en
}

// Only the root node should need to set the wrapper explicitly.
func (en *EmbedNode) RootNodeWrapper(n Node) {
	en.wrapper = n
}

func (en *EmbedNode) Wrapper() Node {
	return en.wrapper
}

// Returns the parent wrapping node if present.
func (en *EmbedNode) Parent() Node {
	if en.parent != nil {
		w := en.parent.wrapper
		if w == nil {
			s := fmt.Sprintf("%s", reflect.TypeOf(en.wrapper))
			panic("parent node without wrapper: en.wrapper is " + s)
		}
		return w
	}
	return nil
}

func (en *EmbedNode) Append(nodes ...Node) {
	for _, n := range nodes {
		en._insertBefore(n, nil)
	}
}

// Next should not be nil. Use Append to add to the end.
func (en *EmbedNode) InsertBefore(n, next Node) {
	// Note that testing next for nil could fail if it is a non-nil interface that is nil, and testing (Node==nil) would be false.
	en._insertBefore(n, next)
}

func (en *EmbedNode) _insertBefore(n, next Node) {
	ne := n.Embed()
	if ne == en {
		panic("inserting into itself")
	}

	// insert in list and get element
	var elem *list.Element
	if next == nil {
		elem = en.childs.PushBack(n)
	} else {
		nexte := next.Embed()
		if nexte.parent != en {
			panic("next is not a child of this node")
		}
		elem = en.childs.InsertBefore(n, nexte.elem)
	}
	if elem == nil {
		panic("element not inserted")
	}

	ne.elem = elem
	ne.parent = en
	ne.wrapper = n // auto set the wrapper

	if en.Hidden() {
		ne.setParentHidden(true)
	}
}

func (en *EmbedNode) Remove(n Node) {
	if !en.HasChild(n) {
		panic("not a child of this node")
	}
	ne := n.Embed()
	en.childs.Remove(ne.elem)
	ne.elem = nil
	ne.parent = nil
	ne.setParentHidden(false)
}

func (en *EmbedNode) FirstChild() Node {
	return en.notHiddenOrNext(en.childs.Front())
}
func (en *EmbedNode) LastChild() Node {
	return en.notHiddenOrPrev(en.childs.Back())
}
func (en *EmbedNode) Prev() Node {
	return en.notHiddenOrPrev(en.elem.Prev())
}
func (en *EmbedNode) Next() Node {
	return en.notHiddenOrNext(en.elem.Next())
}

func (en *EmbedNode) notHiddenOrPrev(e0 *list.Element) Node {
	for e := e0; e != nil; e = e.Prev() {
		n := e.Value.(Node)
		if n.Embed().Hidden() {
			continue
		}
		return n
	}
	return nil
}
func (en *EmbedNode) notHiddenOrNext(e0 *list.Element) Node {
	for e := e0; e != nil; e = e.Next() {
		n := e.Value.(Node)
		if n.Embed().Hidden() {
			continue
		}
		return n
	}
	return nil
}

func (en *EmbedNode) FirstChildInAll() Node {
	return en.elemNode(en.childs.Front())
}
func (en *EmbedNode) LastChildInAll() Node {
	return en.elemNode(en.childs.Back())
}
func (en *EmbedNode) PrevInAll() Node {
	return en.elemNode(en.elem.Prev())
}
func (en *EmbedNode) NextInAll() Node {
	return en.elemNode(en.elem.Next())
}
func (en *EmbedNode) elemNode(e *list.Element) Node {
	if e == nil {
		return nil
	}
	return e.Value.(Node)
}

// Doesn't use Remove/Insert. So implementing nodes overriding those will not see their functions used.
func (en *EmbedNode) Swap(n Node) {
	ne := n.Embed()
	if en.parent != ne.parent {
		panic("nodes don't have the same parent")
	}
	l := &en.parent.childs
	e1 := en.elem
	e2 := ne.elem
	if e1.Next() == e2 {
		l.MoveAfter(e1, e2)
	} else if e2.Next() == e1 {
		l.MoveAfter(e2, e1)
	} else {
		prev := e1.Prev()
		l.MoveAfter(e1, e2)
		if prev == nil {
			l.MoveToFront(e2)
		} else {
			l.MoveAfter(e2, prev)
		}
	}
}

// Current element can be safely removed from inside the loop.
func (en *EmbedNode) IterChildsStop(fn func(Node) bool) {
	var next *list.Element
	for e := en.childs.Front(); e != nil; e = next {
		n := e.Value.(Node)
		next = e.Next()
		if n.Embed().Hidden() {
			continue
		}
		if !fn(n) {
			break
		}
	}
}

// Current element can be safely removed from inside the loop.
func (en *EmbedNode) IterChildsReverseStop(fn func(Node) bool) {
	var prev *list.Element
	for e := en.childs.Back(); e != nil; e = prev {
		n := e.Value.(Node)
		prev = e.Prev()
		if n.Embed().Hidden() {
			continue
		}
		if !fn(n) {
			break
		}
	}
}

func (en *EmbedNode) IterChilds(fn func(Node)) {
	en.IterChildsStop(func(n Node) bool {
		fn(n)
		return true
	})
}
func (en *EmbedNode) IterChildsReverse(fn func(Node)) {
	en.IterChildsReverseStop(func(n Node) bool {
		fn(n)
		return true
	})
}

func (en *EmbedNode) ChildsLen() int {
	c := 0
	for e := en.childs.Back(); e != nil; e = e.Prev() {
		n := e.Value.(Node)
		if n.Embed().Hidden() {
			continue
		}
		c++
	}
	return c
}

func (en *EmbedNode) HasChild(n Node) bool {
	return en == n.Embed().parent
}

func (en *EmbedNode) Expand() (bool, bool) {
	return en.expand.x, en.expand.y
}
func (en *EmbedNode) SetExpand(x, y bool) {
	en.expand.x, en.expand.y = x, y
}
func (en *EmbedNode) Fill() (bool, bool) {
	return en.fill.x, en.fill.y
}
func (en *EmbedNode) SetFill(x, y bool) {
	en.fill.x, en.fill.y = x, y
}

func (en *EmbedNode) hasMarks(m Marks) bool {
	en.marksLock.RLock()
	defer en.marksLock.RUnlock()
	return en.marks.has(m)
}
func (en *EmbedNode) setMarks(m Marks, v bool) {
	en.marksLock.Lock()
	defer en.marksLock.Unlock()
	en.marks.set(m, v)
}

func (en *EmbedNode) NeedsPaint() bool {
	return en.hasMarks(NeedsPaintMark)
}
func (en *EmbedNode) MarkNeedsPaint() {
	en.setNeedsPaint(true)
}
func (en *EmbedNode) setNeedsPaint(v bool) {
	if v && en.NotPaintable() {
		return
	}
	en.setMarks(NeedsPaintMark, v)
	if v {
		//log.Printf("---needspaint %v", reflect.TypeOf(en.wrapper))

		// set mark in parents if not already marked
		for pn := en.parent; pn != nil; pn = pn.parent {
			pn.setMarks(ChildNeedsPaintMark, true)
		}

		// run callbacks with this node rectangle argument
		child := en
		for pn := en.parent; pn != nil; pn = pn.parent {
			pn.wrapper.OnMarkChildNeedsPaint(child.wrapper, &en.Bounds)
			child = pn
		}
	}
}

func (en *EmbedNode) OnMarkChildNeedsPaint(child Node, r *image.Rectangle) {
}

func (en *EmbedNode) ChildNeedsPaint() bool {
	return en.hasMarks(ChildNeedsPaintMark)
}
func (en *EmbedNode) unmarkChildNeedsPaint() {
	en.setMarks(ChildNeedsPaintMark, false)
}

func (en *EmbedNode) Hidden() bool {
	return en.hasMarks(HiddenMark | ParentHiddenMark)
}
func (en *EmbedNode) SetHidden(v bool) {
	en.setMarks(HiddenMark, v)

	// update childs, note that it could have a hidden parent
	isHidden := en.Hidden()
	for e := en.childs.Front(); e != nil; e = e.Next() {
		ce := e.Value.(Node).Embed()
		ce.setParentHidden(isHidden)
	}
}
func (en *EmbedNode) setParentHidden(v bool) {
	en.setMarks(ParentHiddenMark, v)

	// update childs, note that this node itself could be hidden
	isHidden := en.Hidden()
	for e := en.childs.Front(); e != nil; e = e.Next() {
		ce := e.Value.(Node).Embed()
		ce.setParentHidden(isHidden)
	}
}

func (en *EmbedNode) NotDraggable() bool {
	return en.hasMarks(NotDraggableMark)
}
func (en *EmbedNode) SetNotDraggable(v bool) {
	en.setMarks(NotDraggableMark, v)
}
func (en *EmbedNode) PointerInside() bool {
	return en.hasMarks(PointerInsideMark)
}
func (en *EmbedNode) SetPointerInside(v bool) {
	en.setMarks(PointerInsideMark, v)
}
func (en *EmbedNode) NotPaintable() bool {
	return en.hasMarks(NotPaintableMark)
}
func (en *EmbedNode) SetNotPaintable(v bool) {
	en.setMarks(NotPaintableMark, v)
}

func (en *EmbedNode) Measure(hint image.Point) image.Point {
	var max image.Point
	en.IterChilds(func(c Node) {
		m := c.Measure(hint)
		max = MaxPoint(max, m)
	})
	return max
}

func (en *EmbedNode) CalcChildsBounds() {
	en.IterChilds(func(c Node) {
		c.Embed().Bounds = en.Bounds
		c.CalcChildsBounds()
	})
}

func (en *EmbedNode) Paint() {
}

func (en *EmbedNode) OnInputEvent(ev interface{}, p image.Point) bool {
	return false
}

func PaintIfNeeded(node Node, painted func(*image.Rectangle)) {
	if node.Embed().NeedsPaint() {
		if PaintTree(node) {
			painted(&node.Embed().Bounds)
		}
	} else if node.Embed().ChildNeedsPaint() {
		node.Embed().unmarkChildNeedsPaint()
		node.Embed().IterChilds(func(child Node) {
			PaintIfNeeded(child, painted)
		})
	}
}

//var paintDepth int

func PaintTree(node Node) (painted bool) {
	//paintDepth++
	//defer func() { paintDepth-- }()

	ne := node.Embed()
	if ne.Hidden() {
		return false
	}

	//log.Printf("%*s%s", paintDepth, "", reflect.TypeOf(node))

	ne.setNeedsPaint(false)
	ne.unmarkChildNeedsPaint()
	node.Paint()
	node.PaintChilds()
	return true
}

func (en *EmbedNode) PaintChilds() {
	en.IterChilds(func(child Node) {
		_ = PaintTree(child)
	})
}

//func (en *EmbedNode) PaintChilds() {
//	// currently not used: unable to ensure the top layer gets drawn first (multilayer, menu layer)

//	childs := en.Childs()
//	var wg sync.WaitGroup
//	wg.Add(len(childs))
//	for _, child := range childs {
//		go func(child Node) {
//			defer wg.Done()
//			_ = PaintTree(child)
//		}(child)
//	}
//	wg.Wait()
//}

type Marks uint8

const (
	NeedsPaintMark Marks = 1 << iota
	ChildNeedsPaintMark
	PointerInsideMark // mouseEnter/mouseLeave events
	NotDraggableMark
	HiddenMark
	ParentHiddenMark

	// For transparent widgets that cross 2 or more other widgets (ex: separatorHandle). Improves on detecting if others need paint and reduces the number of widgets that get painted.
	NotPaintableMark
)

func (m *Marks) add(u Marks) {
	*m |= u
}
func (m *Marks) remove(u Marks) {
	*m &^= u
}
func (m *Marks) has(u Marks) bool {
	return *m&u > 0
}
func (m *Marks) set(u Marks, v bool) {
	if v {
		m.add(u)
	} else {
		m.remove(u)
	}
}

func MaxPoint(p1, p2 image.Point) image.Point {
	if p1.X < p2.X {
		p1.X = p2.X
	}
	if p1.Y < p2.Y {
		p1.Y = p2.Y
	}
	return p1
}
func MinPoint(p1, p2 image.Point) image.Point {
	if p1.X > p2.X {
		p1.X = p2.X
	}
	if p1.Y > p2.Y {
		p1.Y = p2.Y
	}
	return p1
}
