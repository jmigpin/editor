package widget

import (
	"container/list"
	"image"
	"sync"
)

type Node interface {
	Elem() *list.Element
	SetElem(*list.Element)
	Parent() Node
	SetParent(Node)

	pushBack(Node)           // use package PushBack(...)
	insertBefore(Node, Node) // use package InsertBefore(...)

	Remove(Node)

	FirstChild() Node
	LastChild() Node
	Prev() Node
	Next() Node

	Swap(Node)

	Childs() []Node
	childsList() *list.List
	HasChild(Node) bool

	Expand() (x, y bool)
	SetExpand(x, y bool)
	Fill() (x, y bool)
	SetFill(x, y bool)

	Hidden() bool
	SetHidden(bool)

	Bounds() image.Rectangle
	SetBounds(*image.Rectangle)

	Marks() *Marks
	MarkNeedsPaint()

	Measure(hint image.Point) image.Point

	CalcChildsBounds()

	Paint() // should not be called directly, called by PaintIfNeeded
	PaintChilds()
}

// These cannot be in a specific implementation (like in EmbedNode).
// Allows function overrides (ex: Paint(), Measure(...)) to be called, as well
// as other functions returning the implementing node (ex: Parent())

func PushBack(parent, n Node) {
	n.SetParent(parent)
	parent.pushBack(n)
}

func InsertBefore(parent, n, mark Node) {
	// NOTE: can't test here if mark is nil because it could be receiving
	// a pointer type that is nil, but (Node==nil) will be false.
	// So this function expects the arguments to be present.
	// PushBack is available for use where insertbefore doesn't work.

	if mark.Parent() != parent {
		panic("mark is not a child of this parent")
	}
	n.SetParent(parent)
	parent.insertBefore(n, mark)
}

func AppendChilds(parent Node, nodes ...Node) {
	for _, n := range nodes {
		PushBack(parent, n)
	}
}

func PaintIfNeeded(node Node, painted func(*image.Rectangle)) {
	if node.Marks().NeedsPaint() {
		node.Marks().UnmarkNeedsPaint()
		node.Paint()
		node.PaintChilds()
		b := node.Bounds()
		painted(&b)
	} else if node.Marks().ChildNeedsPaint() {
		node.Marks().UnmarkChildNeedsPaint()
		for _, child := range node.Childs() {
			PaintIfNeeded(child, painted)
		}
	}
}

type EmbedNode struct {
	elem   *list.Element
	parent Node
	childs list.List
	bounds image.Rectangle
	marks  Marks
	expand struct{ x, y bool }
	fill   struct{ x, y bool }

	hidden bool
}

func (en *EmbedNode) Elem() *list.Element {
	return en.elem
}
func (en *EmbedNode) SetElem(elem *list.Element) {
	en.elem = elem
}
func (en *EmbedNode) Parent() Node {
	return en.parent
}
func (en *EmbedNode) SetParent(p Node) {
	en.parent = p
}

func (en *EmbedNode) pushBack(n Node) {
	elem := en.childs.PushBack(n)
	n.SetElem(elem)
}
func (en *EmbedNode) insertBefore(n, mark Node) {
	elem := en.childs.InsertBefore(n, mark.Elem())
	n.SetElem(elem)
}

func (en *EmbedNode) Remove(n Node) {
	en.childs.Remove(n.Elem())
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
		if n.Hidden() {
			continue
		}
		return n
	}
	return nil
}
func (en *EmbedNode) notHiddenOrNext(e0 *list.Element) Node {
	for e := e0; e != nil; e = e.Next() {
		n := e.Value.(Node)
		if n.Hidden() {
			continue
		}
		return n
	}
	return nil
}

func (en *EmbedNode) Swap(n Node) {
	// Doesn't use Remove/Insert. So implementing nodes overriding those will not see their functions used.

	if en.Parent() != n.Parent() {
		panic("nodes don't have the same parent")
	}
	l := en.Parent().childsList()
	e1 := en.Elem()
	e2 := n.Elem()
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
	for e := en.childs.Front(); e != nil; e = e.Next() {
		n := e.Value.(Node)
		if n.Hidden() {
			continue
		}
		u = append(u, n)
	}
	return u
}
func (en *EmbedNode) childsList() *list.List {
	return &en.childs
}
func (en *EmbedNode) HasChild(n Node) bool {
	for e := en.childs.Front(); e != nil; e = e.Next() {
		w := e.Value.(Node)
		if n == w {
			return true
		}
	}
	return false
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
func (en *EmbedNode) Hidden() bool {
	if en.hidden {
		return true
	}
	if en.Parent() != nil {
		return en.Parent().Hidden()
	}
	return false
}
func (en *EmbedNode) SetHidden(v bool) {
	en.hidden = v
}

func (en *EmbedNode) Bounds() image.Rectangle {
	return en.bounds
}
func (en *EmbedNode) SetBounds(b *image.Rectangle) {
	en.bounds = *b
}

func (en *EmbedNode) Marks() *Marks {
	return &en.marks
}
func (en *EmbedNode) MarkNeedsPaint() {
	en.marks.MarkNeedsPaint()
	// set mark in parents
	for n := en.Parent(); n != nil; n = n.Parent() {
		n.Marks().MarkChildNeedsPaint()
	}
}

func (en *EmbedNode) Paint() {
}

func (en *EmbedNode) PaintChilds() {
	en.Marks().UnmarkChildNeedsPaint()
	var wg sync.WaitGroup
	for _, child := range en.Childs() {
		wg.Add(1)
		go func(child Node) {
			defer wg.Done()
			en.Marks().UnmarkNeedsPaint()
			child.Paint()
			child.PaintChilds()
		}(child)
	}
	wg.Wait()
}

//type LeafNode struct {
//	EmbedNode
//}

//func (ln *LeafNode) pushBack(n Node) {
//	panic("can't insert child on a leaf node")
//}
//func (ln *LeafNode) insertBefore(n, mark Node) {
//	panic("can't insert child on a leaf node")
//}
//func (ln *LeafNode) CalcChildsBounds() {
//}

//type ShellNode struct {
//	EmbedNode
//}

//func (sn *ShellNode) pushBack(n Node) {
//	if sn.NChilds() > 0 {
//		panic("shell node already has a child")
//	}
//	sn.EmbedNode.pushBack(n)
//}
//func (sn *ShellNode) insertBefore(n, mark Node) {
//	panic("shell node can have only one child, use pushback")
//}
//func (sn *ShellNode) Measure(hint image.Point) *image.Point {
//	return sn.FirstChild().Measure(hint)
//}
//func (sn *ShellNode) CalcChildsBounds() {
//	sn.FirstChild().SetBounds(sn.Bounds())
//	sn.FirstChild().CalcChildsBounds()
//}

//type ContainerNode struct {
//	EmbedNode
//}

type Marks uint8

const (
	MarkNeedsPaint Marks = 1 << iota
	MarkChildNeedsPaint
)

func (m *Marks) Add(u Marks) {
	*m |= u
}
func (m *Marks) Remove(u Marks) {
	*m &^= u
}
func (m *Marks) Has(u Marks) bool {
	return *m&u > 0
}

func (m *Marks) NeedsPaint() bool      { return m.Has(MarkNeedsPaint) }
func (m *Marks) ChildNeedsPaint() bool { return m.Has(MarkChildNeedsPaint) }

func (m *Marks) MarkNeedsPaint()      { m.Add(MarkNeedsPaint) }
func (m *Marks) MarkChildNeedsPaint() { m.Add(MarkChildNeedsPaint) }

func (m *Marks) UnmarkNeedsPaint()      { m.Remove(MarkNeedsPaint) }
func (m *Marks) UnmarkChildNeedsPaint() { m.Remove(MarkChildNeedsPaint) }

func IsAPrevOf(a, b Node) bool {
	for u := b.Prev(); u != nil; u = u.Prev() {
		if u == a {
			return true
		}
	}
	return false
}
func IsANextOf(a, b Node) bool {
	for u := b.Next(); u != nil; u = u.Next() {
		if u == a {
			return true
		}
	}
	return false
}
