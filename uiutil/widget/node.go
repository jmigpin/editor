package widget

import (
	"container/list"
	"image"
	"sync"
)

type Node interface {
	Wrappee() *EmbedNode

	Parent() Node // returns wrapper if present

	PushBack(n Node)
	InsertBefore(n, mark Node)
	Append(n ...Node)
	Remove(Node)

	FirstChild() Node
	LastChild() Node
	Prev() Node
	Next() Node

	Swap(Node)

	Childs() []Node // doesn't include hidden nodes

	Expand() (x, y bool)
	SetExpand(x, y bool)
	Fill() (x, y bool)
	SetFill(x, y bool)

	Bounds() image.Rectangle
	SetBounds(*image.Rectangle)

	Marks() *Marks
	MarkNeedsPaint()
	Hidden() bool
	SetHidden(bool)

	Measure(hint image.Point) image.Point
	CalcChildsBounds()
	Paint()
	PaintChilds()

	OnInputEvent(ev interface{}, p image.Point) bool
}

type EmbedNode struct {
	childs  list.List
	elem    *list.Element
	parent  *EmbedNode
	wrapper Node
	bounds  image.Rectangle
	marks   Marks
	expand  struct{ x, y bool }
	fill    struct{ x, y bool }

	marksLock sync.Mutex
}

func (en *EmbedNode) Wrappee() *EmbedNode {
	return en
}

// Important when a node needs to reach a wrap implementation.
func (en *EmbedNode) SetWrapper(n Node) {
	if en != n.Wrappee() {
		panic("node not wrapping")
	}
	en.wrapper = n
}

// Returns the parent wrapping node if present.
func (en *EmbedNode) Parent() Node {
	if en.parent != nil {
		w := en.parent.wrapper
		if w != nil {
			return w
		}
		return en.parent
	}
	return nil
}

func (en *EmbedNode) PushBack(n Node) {
	nw := n.Wrappee()
	if nw == en {
		panic("!")
	}
	nw.parent = en
	elem := en.childs.PushBack(n)
	if elem == nil {
		panic("element not inserted")
	}
	nw.elem = elem
	if en.hasHiddenParent() {
		nw.Marks().SetParentHidden(true)
	}
}

// Note that the next node can't be tested for nil because it could be receiving a non-nil interface that is nil, and testing (Node==nil) would be false. So this function expects the arguments to be present and have PushBack be used for appends.
func (en *EmbedNode) InsertBefore(n, next Node) {
	nextw := next.Wrappee()
	if nextw.parent != en {
		panic("next is not a child of this node")
	}
	nw := n.Wrappee()
	if nw == en {
		panic("!")
	}
	nw.parent = en
	elem := en.childs.InsertBefore(n, next.Wrappee().elem)
	if elem == nil {
		panic("element not inserted")
	}
	nw.elem = elem
	if en.hasHiddenParent() {
		nw.Marks().SetParentHidden(true)
	}
}
func (en *EmbedNode) Append(nodes ...Node) {
	for _, n := range nodes {
		en.PushBack(n)
	}
}

func (en *EmbedNode) Remove(n Node) {
	nw := n.Wrappee()
	en.childs.Remove(nw.elem)
	nw.parent = nil
	nw.Marks().SetParentHidden(false)
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

// Doesn't use Remove/Insert. So implementing nodes overriding those will not see their functions used.
func (en *EmbedNode) Swap(n Node) {
	nw := n.Wrappee()
	if en.parent != nw.parent {
		panic("nodes don't have the same parent")
	}
	l := &en.parent.childs
	e1 := en.elem
	e2 := nw.elem
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
		u = append(u, n)
	}
	return u
}

func (en *EmbedNode) HasChild(n Node) bool {
	return en == n.Wrappee().parent
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

func (en *EmbedNode) Marks() *Marks {
	return &en.marks
}
func (en *EmbedNode) MarkNeedsPaint() {
	en.marks.SetNeedsPaint(true)
	// set mark in parents
	for n := en.parent; n != nil; n = n.parent {
		m := n.Marks()
		if m.ChildNeedsPaint() {
			break // parents already marked, early exit
		}
		m.SetChildNeedsPaint(true)
	}
}

func (en *EmbedNode) Hidden() bool {
	return en.marks.Hidden() || en.marks.ParentHidden()
}
func (en *EmbedNode) SetHidden(v bool) {
	en.marks.SetHidden(v)
	en.setParentHiddenInChilds(v)
}
func (en *EmbedNode) setParentHiddenInChilds(v bool) {
	for e := en.childs.Front(); e != nil; e = e.Next() {
		cw := e.Value.(Node).Wrappee()
		cw.Marks().SetParentHidden(v)
		cw.setParentHiddenInChilds(v)
	}
}
func (en *EmbedNode) hasHiddenParent() bool {
	if en.parent != nil {
		m := en.parent.Marks()
		return m.Hidden() || m.ParentHidden()
	}
	return false
}

func (en *EmbedNode) Measure(hint image.Point) image.Point {
	return image.Point{}
}
func (en *EmbedNode) CalcChildsBounds() {
}
func (en *EmbedNode) Paint() {
}

func (en *EmbedNode) PaintChilds() {
	en.marksLock.Lock()
	en.marks.SetChildNeedsPaint(false)
	en.marksLock.Unlock()
	var wg sync.WaitGroup
	for _, child := range en.Childs() {
		wg.Add(1)
		go func(child Node) {
			defer wg.Done()
			en.marksLock.Lock()
			en.marks.SetNeedsPaint(false)
			en.marksLock.Unlock()
			child.Paint()
			child.PaintChilds()
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
	u := sn.Bounds()
	sn.FirstChild().SetBounds(&u)
	sn.FirstChild().CalcChildsBounds()
}

type Marks uint8

const (
	MarkNeedsPaint Marks = 1 << iota
	MarkChildNeedsPaint
	MarkPointerInside // used for mouseEnter/mouseLeave events
	MarkNotDraggable
	MarkHidden
	MarkParentHidden
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
func (m *Marks) Set(u Marks, v bool) {
	if v {
		m.Add(u)
	} else {
		m.Remove(u)
	}
}

func (m *Marks) NeedsPaint() bool     { return m.Has(MarkNeedsPaint) }
func (m *Marks) SetNeedsPaint(v bool) { m.Set(MarkNeedsPaint, v) }

func (m *Marks) ChildNeedsPaint() bool     { return m.Has(MarkChildNeedsPaint) }
func (m *Marks) SetChildNeedsPaint(v bool) { m.Set(MarkChildNeedsPaint, v) }

func (m *Marks) PointerInside() bool     { return m.Has(MarkPointerInside) }
func (m *Marks) SetPointerInside(v bool) { m.Set(MarkPointerInside, v) }

func (m *Marks) NotDraggable() bool     { return m.Has(MarkNotDraggable) }
func (m *Marks) SetNotDraggable(v bool) { m.Set(MarkNotDraggable, v) }

func (m *Marks) Hidden() bool     { return m.Has(MarkHidden) }
func (m *Marks) SetHidden(v bool) { m.Set(MarkHidden, v) }

func (m *Marks) ParentHidden() bool     { return m.Has(MarkParentHidden) }
func (m *Marks) SetParentHidden(v bool) { m.Set(MarkParentHidden, v) }
