package widget

import (
	"container/list"
	"image"
	"image/color"
	"strings"

	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/uiutil/event"
)

type Node interface {
	fullNode() // ensure that EmbNode can't be directly assigned to a Node

	Embed() *EmbedNode

	InsertBefore(n Node, mark *EmbedNode)
	Append(n ...Node)
	Remove(child Node)

	Measure(hint image.Point) image.Point

	LayoutMarked()
	LayoutTree()
	Layout() // set childs bounds, don't call childs layout
	ChildsLayoutTree()

	PaintMarked() image.Rectangle
	PaintTree() bool
	PaintBase() // pre-paint step, useful for widgets with a pre-paint stage
	Paint()
	ChildsPaintTree()

	OnThemeChange()
	OnChildMarked(child Node, newMarks Marks)
	OnInputEvent(ev interface{}, p image.Point) event.Handle
}

//----------

// Doesn't allow embed to be assigned to a Node directly, which prevents a range of programming mistakes. This is the node other widgets should inherit from.
type ENode struct {
	EmbedNode
}

func (ENode) fullNode() {}

//----------

type EmbedNode struct {
	Bounds  image.Rectangle
	Cursor  Cursor
	Marks   Marks
	Wrapper Node
	Parent  *EmbedNode

	childs list.List
	elem   *list.Element

	theme Theme
}

//----------

func (en *EmbedNode) Embed() *EmbedNode {
	return en
}

// Only the root node should need to set the wrapper explicitly.
func (en *EmbedNode) SetWrapperForRoot(n Node) {
	en.Wrapper = n
}

//----------

// If a node wants its InsertBefore implementation to be used, the wrapper must be set.
func (en *EmbedNode) Append(nodes ...Node) {
	for _, n := range nodes {
		if en.Wrapper != nil {
			en.Wrapper.InsertBefore(n, nil)
		} else {
			en.InsertBefore(n, nil)
		}
	}
}

func (en *EmbedNode) InsertBefore(child Node, next *EmbedNode) {
	childe := child.Embed()

	if childe == en {
		panic("inserting into itself")
	}
	if childe.Parent != nil {
		panic("element already has a parent")
	}

	// insert in list and get element
	var elem *list.Element
	if next == nil {
		elem = en.childs.PushBack(childe)
	} else {
		// ensure next element is a child of this node
		if next.Parent != en {
			panic("next is not a child of this node")
		}

		elem = en.childs.InsertBefore(childe, next.elem)
	}
	if elem == nil {
		panic("element not inserted")
	}

	childe.elem = elem
	childe.Parent = en
	childe.Wrapper = child // auto set the wrapper

	en.MarkNeedsLayoutAndPaint()

	childe.themeChangeCallback()
}

//----------

func (en *EmbedNode) Remove(n Node) {
	ne := n.Embed()
	if ne.Parent != en {
		panic("not a child of this node")
	}
	en.childs.Remove(ne.elem)
	ne.elem = nil
	ne.Parent = nil

	en.MarkNeedsLayoutAndPaint()
}

//----------

// Doesn't use Remove/Insert. So implementing nodes overriding those will not see their functions used.
func (en *EmbedNode) Swap(u Node) {
	eu := u.Embed()
	if en.Parent != eu.Parent {
		panic("nodes don't have the same parent")
	}
	l := &en.Parent.childs
	e1 := en.elem
	e2 := eu.elem
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

//----------

func (en *EmbedNode) ChildsLen() int {
	return en.childs.Len()
}

//----------

func elemEmbed(e *list.Element) *EmbedNode {
	if e == nil {
		return nil
	}
	return e.Value.(*EmbedNode)
}
func elemWrapper(e *list.Element) Node {
	if e == nil {
		return nil
	}
	return e.Value.(*EmbedNode).Wrapper
}

//----------

func (en *EmbedNode) FirstChild() *EmbedNode {
	return elemEmbed(en.childs.Front())
}
func (en *EmbedNode) LastChild() *EmbedNode {
	return elemEmbed(en.childs.Back())
}
func (en *EmbedNode) NextSibling() *EmbedNode {
	return elemEmbed(en.elem.Next())
}
func (en *EmbedNode) PrevSibling() *EmbedNode {
	return elemEmbed(en.elem.Prev())
}

//----------

func (en *EmbedNode) FirstChildWrapper() Node {
	return elemWrapper(en.childs.Front())
}
func (en *EmbedNode) LastChildWrapper() Node {
	return elemWrapper(en.childs.Back())
}
func (en *EmbedNode) NextSiblingWrapper() Node {
	return elemWrapper(en.elem.Next())
}
func (en *EmbedNode) PrevSiblingWrapper() Node {
	return elemWrapper(en.elem.Prev())
}

//----------

func (en *EmbedNode) Iterate(f func(*EmbedNode) bool) {
	for e := en.childs.Front(); e != nil; e = e.Next() {
		if !f(elemEmbed(e)) {
			break
		}
	}
}
func (en *EmbedNode) IterateReverse(f func(*EmbedNode) bool) {
	for e := en.childs.Back(); e != nil; e = e.Prev() {
		if !f(elemEmbed(e)) {
			break
		}
	}
}
func (en *EmbedNode) IterateWrappers(f func(Node) bool) {
	for e := en.childs.Front(); e != nil; e = e.Next() {
		if !f(elemWrapper(e)) {
			break
		}
	}
}
func (en *EmbedNode) IterateWrappersReverse(f func(Node) bool) {
	for e := en.childs.Back(); e != nil; e = e.Prev() {
		if !f(elemWrapper(e)) {
			break
		}
	}
}

//----------

func (en *EmbedNode) Iterate2(f func(*EmbedNode)) {
	for e := en.childs.Front(); e != nil; e = e.Next() {
		f(elemEmbed(e))
	}
}
func (en *EmbedNode) IterateReverse2(f func(*EmbedNode)) {
	for e := en.childs.Back(); e != nil; e = e.Prev() {
		f(elemEmbed(e))
	}
}
func (en *EmbedNode) IterateWrappers2(f func(Node)) {
	for e := en.childs.Front(); e != nil; e = e.Next() {
		f(elemWrapper(e))
	}
}
func (en *EmbedNode) IterateWrappersReverse2(f func(Node)) {
	for e := en.childs.Back(); e != nil; e = e.Prev() {
		f(elemWrapper(e))
	}
}

//----------

func (en *EmbedNode) ChildsWrappers() []Node {
	w := []Node{}
	en.IterateWrappers2(func(c Node) {
		w = append(w, c)
	})
	return w
}

//----------

func (en *EmbedNode) MarkUp(m Marks) {
	en.markUp(m, true)
}

func (en *EmbedNode) markUp(m Marks, makeCallback bool) {
	old := en.Marks
	en.Marks |= m
	changed := en.Marks ^ old
	if changed != 0 {
		if en.Parent != nil {

			// update marks to add to parent
			u := changed
			if u.HasAny(MarkNeedsPaint) {
				u.Remove(MarkNeedsPaint)
				u.Add(MarkChildNeedsPaint)
			}
			if u.HasAny(MarkNeedsLayout) {
				u.Remove(MarkNeedsLayout)
				u.Add(MarkChildNeedsLayout)
			}

			// mark parent
			en.Parent.markUp(u, makeCallback)

			if makeCallback && en.Parent.Wrapper != nil {
				en.Parent.Wrapper.OnChildMarked(en.Wrapper, changed)
			}
		}
	}

}

func (en *EmbedNode) OnChildMarked(child Node, newMarks Marks) {
}

//----------

func (en *EmbedNode) MarkNeedsLayout() {
	en.MarkUp(MarkNeedsLayout)
}
func (en *EmbedNode) MarkNeedsPaint() {
	en.MarkUp(MarkNeedsPaint)
}
func (en *EmbedNode) MarkNeedsLayoutAndPaint() {
	en.MarkUp(MarkNeedsLayout | MarkNeedsPaint)
}

//----------

func (en *EmbedNode) TreeNeedsPaint() bool {
	return en.Marks.HasAny(MarkNeedsPaint | MarkChildNeedsPaint)
}

func (en *EmbedNode) TreeNeedsLayout() bool {
	return en.Marks.HasAny(MarkNeedsLayout | MarkChildNeedsLayout)
}

//----------

func (en *EmbedNode) Measure(hint image.Point) image.Point {
	var max image.Point
	en.IterateWrappers2(func(c Node) {
		m := c.Measure(hint)
		max = imageutil.MaxPoint(max, m)
	})
	return max
}

//----------

func (en *EmbedNode) LayoutMarked() {
	if en.Marks.HasAny(MarkNeedsLayout) {
		en.Wrapper.LayoutTree()
	} else if en.Marks.HasAny(MarkChildNeedsLayout) {
		en.Marks.Remove(MarkChildNeedsLayout)
		en.IterateWrappers2(func(c Node) {
			c.LayoutMarked()
		})
	}
}

//var depth int

func (en *EmbedNode) LayoutTree() {
	//fmt.Printf("%*s layouttree %T %v\n", depth*4, "", en.Wrapper, en.Bounds)
	//depth++
	//defer func() { depth-- }()

	en.Marks.Remove(MarkNeedsLayout | MarkChildNeedsLayout)

	// keep/set default bounds before layouting childs
	cbm := map[*EmbedNode]image.Rectangle{}
	en.Iterate2(func(c *EmbedNode) {
		cbm[c] = c.Bounds
		c.Bounds = en.Bounds // parent bounds

		// set to empty if not visible
		if c.Marks.HasAny(MarkForceZeroBounds) {
			c.Bounds = image.Rectangle{}
		}
	})

	en.Wrapper.Layout()
	en.Wrapper.ChildsLayoutTree()

	// auto detect if it needs paint if bounds change
	en.Iterate2(func(c *EmbedNode) {
		if cb, ok := cbm[c]; ok && c.Bounds != cb {
			c.MarkNeedsPaint()
		}
	})
}

func (en *EmbedNode) Layout() {
}

func (en *EmbedNode) ChildsLayoutTree() {
	en.IterateWrappers2(func(c Node) {
		c.LayoutTree()
	})
}

//----------

func (en *EmbedNode) PaintMarked() image.Rectangle {
	u := image.Rectangle{}

	if en.Marks.HasAny(MarkNeedsPaint) {
		if en.Wrapper.PaintTree() {
			u = u.Union(en.Bounds)
		}
	} else if en.Marks.HasAny(MarkChildNeedsPaint) {
		en.Marks.Remove(MarkChildNeedsPaint)
		en.IterateWrappers2(func(c Node) {
			r := c.PaintMarked()
			u = u.Union(r)
		})
	}

	return u
}

func (en *EmbedNode) PaintTree() bool {
	en.Marks.Remove(MarkNeedsPaint | MarkChildNeedsPaint)

	if en.Marks.HasAny(MarkNotPaintable | MarkForceZeroBounds) {
		return false
	}

	en.Wrapper.PaintBase()
	en.Wrapper.Paint()
	en.Wrapper.ChildsPaintTree()
	return true
}

func (en *EmbedNode) PaintBase() {
}

func (en *EmbedNode) Paint() {
}

func (en *EmbedNode) ChildsPaintTree() {
	en.IterateWrappers2(func(c Node) {
		c.PaintTree()
	})
}

//----------

func (en *EmbedNode) OnInputEvent(ev interface{}, p image.Point) event.Handle {
	return event.NotHandled
}

//----------

func (en *EmbedNode) SetTheme(t Theme) {
	defer en.themeChangeCallback()
	defer en.MarkNeedsPaint()  // possible palette change/update
	defer en.MarkNeedsLayout() // possible font change

	en.theme = t
}

func (en *EmbedNode) Theme() *Theme {
	return &en.theme
}

//----------

func (en *EmbedNode) ThemePalette() Palette {
	return en.theme.Palette
}

func (en *EmbedNode) SetThemePalette(p Palette) {
	defer en.themeChangeCallback()
	defer en.MarkNeedsPaint()

	en.theme.SetPalette(p)
}

func (en *EmbedNode) SetThemePaletteColor(name string, c color.Color) {
	defer en.themeChangeCallback()
	defer en.MarkNeedsPaint()

	en.theme.SetPaletteColor(name, c)
}

func (en *EmbedNode) SetThemePaletteNamePrefix(prefix string) {
	defer en.themeChangeCallback()
	defer en.MarkNeedsPaint()

	en.theme.SetPaletteNamePrefix(prefix)
}

//----------

func (en *EmbedNode) TreeThemePaletteColor(name string) color.Color {
	if c, ok := en.treeThemePaletteColor2(name); ok {
		return c
	}
	// last resort: a color that is not white/black to help debug
	return cint(0xff0000)
}

func (en *EmbedNode) treeThemePaletteColor2(name string) (color.Color, bool) {
	if !strings.HasPrefix(name, en.theme.PaletteNamePrefix) {
		s := en.theme.PaletteNamePrefix + name
		if c, ok := en.treeThemePaletteColor2(s); ok {
			return c, true
		}
	}
	if c, ok := en.theme.Palette[name]; ok {
		return c, true
	}
	if en.Parent != nil {
		if c, ok := en.Parent.treeThemePaletteColor2(name); ok {
			return c, true
		}
	}
	// at root tree (parent is nil) and not found, try default palette
	if c, ok := DefaultPalette[name]; ok {
		return c, true
	}
	return nil, false
}

//----------

func (en *EmbedNode) SetThemeFont(f ThemeFont) {
	defer en.themeChangeCallback()
	defer en.MarkNeedsLayout()

	en.theme.SetFont(f)
}

func (en *EmbedNode) TreeThemeFont() ThemeFont {
	for n := en; n != nil; n = n.Parent {
		if n.theme.Font != nil {
			return n.theme.Font
		}
	}
	return DefaultThemeFont() // TODO: instance that gets updated?
}

//----------

func (en *EmbedNode) themeChangeCallback() {
	if en.Wrapper != nil {
		en.Wrapper.OnThemeChange()
	}
	en.Iterate2(func(c *EmbedNode) {
		c.themeChangeCallback()
	})
}

func (en *EmbedNode) OnThemeChange() {
}

//----------

type Marks uint16

func (m *Marks) Add(u Marks)        { *m |= u }
func (m *Marks) Remove(u Marks)     { *m &^= u }
func (m Marks) Mask(u Marks) Marks  { return m & u }
func (m Marks) HasAny(u Marks) bool { return m.Mask(u) > 0 }

//func (m *Marks) Modify(u Marks, v bool) {
//	if v {
//		m.Add(u)
//	} else {
//		m.Remove(u)
//	}
//}
//func (m Marks) Changes(u Marks) Marks {
//	old := m
//	m |= u
//	return m ^ old
//}

//----------

const (
	MarkNeedsPaint Marks = 1 << iota
	MarkNeedsLayout

	MarkChildNeedsPaint
	MarkChildNeedsLayout

	//MarkHidden
	//MarkParentHidden

	MarkPointerInside // mouseEnter/mouseLeave events
	MarkNotDraggable  // won't emit mouseDrag events

	MarkForceZeroBounds // sets bounds to zero (aka not visible)

	MarkInBoundsHandlesEvent // helps with layer nodes keep events

	// For transparent widgets that cross two or more other widgets (ex: a non visible separator handle). Improves on detecting if others need paint.
	MarkNotPaintable
)
