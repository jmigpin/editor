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

	Parent() Node // returns wrapper if present

	PushBack(n Node)
	InsertBefore(n, mark Node)
	Append(n ...Node)
	Remove(Node)

	FirstChild() Node // doesn't include hidden nodes
	LastChild() Node  // doesn't include hidden nodes
	Prev() Node       // doesn't include hidden nodes
	Next() Node       // doesn't include hidden nodes

	Swap(Node)

	Childs() []Node // doesn't include hidden nodes
	AllChilds() []Node

	Bounds() image.Rectangle
	SetBounds(*image.Rectangle)

	OnMarkChildNeedsPaint(Node, *image.Rectangle)

	Measure(hint image.Point) image.Point
	CalcChildsBounds()
	Paint()
	PaintChilds()

	OnInputEvent(ev interface{}, p image.Point) bool
}

type EmbedNode struct {
	childs    list.List
	elem      *list.Element
	parent    *EmbedNode
	wrapper   Node
	bounds    image.Rectangle
	expand    struct{ x, y bool }
	fill      struct{ x, y bool }
	cursorRef *CursorRef

	marks     Marks
	marksLock sync.RWMutex
}

func (en *EmbedNode) Embed() *EmbedNode {
	return en
}

func (en *EmbedNode) SetPointerCursor(ctx Context, c Cursor) {
	CursorStk.Pop(en.cursorRef)
	en.cursorRef = CursorStk.Push(c)
	CursorStk.SetTop(ctx)
}

func (en *EmbedNode) UnsetPointerCursor(ctx Context) {
	if en.cursorRef != nil {
		CursorStk.Pop(en.cursorRef)
		en.cursorRef = nil
		CursorStk.SetTop(ctx)
	}
}

// Important when a node needs to reach a wrap implementation.
func (en *EmbedNode) SetWrapper(n Node) {
	if en != n.Embed() {
		panic("node not wrapping")
	}
	en.wrapper = n
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

func (en *EmbedNode) PushBack(n Node) {
	ne := n.Embed()
	if ne == en {
		panic("inserting into itself")
	}
	elem := en.childs.PushBack(n)
	if elem == nil {
		panic("element not inserted")
	}
	ne.elem = elem
	ne.parent = en
	if en.Hidden() {
		ne.setParentHidden(true)
	}
}

func (en *EmbedNode) InsertBefore(n, next Node) {
	// This function expects the arguments to be present and have PushBack be used for appends.
	// Note that the next node can't be tested for nil because it could be receiving a non-nil interface that is nil, and testing (Node==nil) would be false.

	nexte := next.Embed()
	if nexte.parent != en {
		panic("next is not a child of this node")
	}
	ne := n.Embed()
	if ne == en {
		panic("inserting into itself")
	}
	elem := en.childs.InsertBefore(n, nexte.elem)
	if elem == nil {
		panic("element not inserted")
	}
	ne.elem = elem
	ne.parent = en
	if en.Hidden() {
		ne.setParentHidden(true)
	}
}
func (en *EmbedNode) Append(nodes ...Node) {
	for _, n := range nodes {
		en.PushBack(n)
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

func (en *EmbedNode) Childs() []Node {
	var u []Node
	for n := en.FirstChild(); n != nil; n = n.Next() {
		if !n.Embed().Hidden() {
			u = append(u, n)
		}
	}
	return u
}
func (en *EmbedNode) AllChilds() []Node {
	var u []Node
	for e := en.childs.Front(); e != nil; e = e.Next() {
		n := e.Value.(Node)
		u = append(u, n)
	}
	return u
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

func (en *EmbedNode) Bounds() image.Rectangle {
	return en.bounds
}
func (en *EmbedNode) SetBounds(b *image.Rectangle) {
	en.bounds = *b
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
	en.SetNeedsPaint(true)
}
func (en *EmbedNode) SetNeedsPaint(v bool) {
	en.setMarks(NeedsPaintMark, v)
	if v {
		// set mark in parents
		r := en.Bounds()
		child := en.wrapper
		for n := en.Parent(); n != nil; n = n.Parent() {
			n.Embed().setMarks(ChildNeedsPaintMark, true)
			n.OnMarkChildNeedsPaint(child, &r)
			child = n
		}
	}
}

func (en *EmbedNode) ChildNeedsPaint() bool {
	return en.hasMarks(ChildNeedsPaintMark)
}
func (en *EmbedNode) UnmarkChildNeedsPaint() {
	en.setMarks(ChildNeedsPaintMark, false)
}

// Child is an immediate child of this node, while the rectangle is the bounds of the original node that requested paint.
func (en *EmbedNode) OnMarkChildNeedsPaint(child Node, r *image.Rectangle) {
}

func (en *EmbedNode) Hidden() bool {
	return en.hasMarks(HiddenMark | ParentHiddenMark)
}
func (en *EmbedNode) SetHidden(v bool) {
	en.setMarks(HiddenMark, v)
	for e := en.childs.Front(); e != nil; e = e.Next() {
		ce := e.Value.(Node).Embed()
		ce.setParentHidden(v)
	}
}
func (en *EmbedNode) setParentHidden(v bool) {
	en.setMarks(ParentHiddenMark, v)
	for e := en.childs.Front(); e != nil; e = e.Next() {
		ce := e.Value.(Node).Embed()
		ce.setParentHidden(v)
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

func PaintTree(node Node) (painted bool) {
	ne := node.Embed()
	ne.SetNeedsPaint(false)
	if !node.Embed().Hidden() {
		node.Paint()
		painted = true
	}
	node.PaintChilds()
	return
}

//func (en *EmbedNode) PaintChilds() {
//	en.UnmarkChildNeedsPaint()
//	for _, child := range en.Childs() {
//		_ = PaintTree(child)
//	}
//}

func (en *EmbedNode) PaintChilds() {
	childs := en.Childs()
	var wg sync.WaitGroup
	wg.Add(len(childs))
	en.UnmarkChildNeedsPaint()
	for _, child := range childs {
		go func(child Node) {
			defer wg.Done()
			_ = PaintTree(child)
		}(child)
	}
	wg.Wait()
}

func (en *EmbedNode) OnInputEvent(ev interface{}, p image.Point) bool {
	return false
}

type LeafEmbedNode struct {
	EmbedNode
}

func (ln *LeafEmbedNode) PushBack(n Node) {
	panic("can't insert child on a leaf node")
}
func (ln *LeafEmbedNode) InsertBefore(n, mark Node) {
	panic("can't insert child on a leaf node")
}
func (ln *LeafEmbedNode) CalcChildsBounds() {
}

type ShellEmbedNode struct {
	EmbedNode
}

func (sn *ShellEmbedNode) PushBack(n Node) {
	if sn.FirstChild() != nil {
		panic("shell node already has a child")
	}
	sn.EmbedNode.PushBack(n)
}
func (sn *ShellEmbedNode) InsertBefore(n, mark Node) {
	panic("shell node can have only one child, use pushback")
}
func (sn *ShellEmbedNode) Measure(hint image.Point) image.Point {
	return sn.FirstChild().Measure(hint)
}
func (sn *ShellEmbedNode) CalcChildsBounds() {
	b := sn.Bounds()
	child := sn.FirstChild()
	child.SetBounds(&b)
	child.CalcChildsBounds()
}
func (sn *ShellEmbedNode) Paint() {
}

type ContainerEmbedNode struct {
	EmbedNode
}

func (cn *ContainerEmbedNode) Paint() {
}

type Marks uint8

const (
	NeedsPaintMark Marks = 1 << iota
	ChildNeedsPaintMark
	PointerInsideMark // mouseEnter/mouseLeave events
	NotDraggableMark
	HiddenMark
	ParentHiddenMark
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
