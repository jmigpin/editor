package uiutil

import (
	"container/list"
	"image"
	"sync"
)

type Container struct {
	Owner interface{} // owner data - external use

	Bounds image.Rectangle

	childs list.List
	elem   *list.Element

	Parent *Container

	needMeasure     bool
	needPaint       bool
	childNeedsPaint bool

	PaintFunc  func()
	OnCalcFunc func()

	Style Style
}

func (c *Container) NChilds() int {
	return c.childs.Len()
}
func (c *Container) FirstChild() *Container {
	e := c.childs.Front()
	if e == nil {
		return nil
	}
	return e.Value.(*Container)
}
func (c *Container) LastChild() *Container {
	e := c.childs.Back()
	if e == nil {
		return nil
	}
	return e.Value.(*Container)
}
func (c *Container) PrevSibling() *Container {
	e := c.elem.Prev()
	if e == nil {
		return nil
	}
	return e.Value.(*Container)
}
func (c *Container) NextSibling() *Container {
	e := c.elem.Next()
	if e == nil {
		return nil
	}
	return e.Value.(*Container)
}

// if next is nil it appends to the end.
func (c *Container) InsertChildBefore(c2, next *Container) {
	c2.Parent = c
	if next == nil {
		c2.elem = c.childs.PushBack(c2)
	} else {
		if next.Parent != c {
			panic("element is not a child of this container")
		}
		if next.elem == nil {
			panic("next elem nil")
		}
		c2.elem = c.childs.InsertBefore(c2, next.elem)
	}
}
func (c *Container) RemoveChild(c2 *Container) {
	c.childs.Remove(c2.elem)
}

func (c *Container) AppendChilds(cs ...*Container) {
	for _, c2 := range cs {
		c.InsertChildBefore(c2, nil)
	}
}
func (c *Container) Childs() []*Container {
	u := make([]*Container, 0, c.childs.Len())
	for e := c.childs.Front(); e != nil; e = e.Next() {
		w := e.Value.(*Container)
		u = append(u, w)
	}
	return u
}

func (c *Container) IsAPrevSiblingOf(c2 *Container) bool {
	for u := c2.elem.Prev(); u != nil; u = u.Prev() {
		if u == c.elem {
			return true
		}
	}
	return false
}
func (c *Container) IsANextSiblingOf(c2 *Container) bool {
	for u := c2.elem.Next(); u != nil; u = u.Next() {
		if u == c.elem {
			return true
		}
	}
	return false
}

func (c *Container) SwapWithSibling(c2 *Container) {
	if c.Parent != c2.Parent {
		panic("containers don't have the same parent")
	}

	l := &c.Parent.childs // need to get pointer, a list copy won't work!

	e1 := c.elem
	e2 := c2.elem
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

func (c *Container) CalcChildsBounds() {
	SimpleBoxModelCalcChildsBounds(c)
}

func (c *Container) paint() {
	c.needPaint = false
	c.childNeedsPaint = false
	if c.PaintFunc != nil {
		c.PaintFunc()
	}
	var wg sync.WaitGroup
	for _, child := range c.Childs() {
		wg.Add(1)
		go func(child *Container) {
			defer wg.Done()
			child.paint()
		}(child)
	}
	wg.Wait()
}
func (c *Container) PaintIfNeeded(cb func(*image.Rectangle)) {
	if c.needPaint {
		c.paint()
		cb(&c.Bounds) // section that needs update
	} else if c.childNeedsPaint {
		c.childNeedsPaint = false
		for _, child := range c.Childs() {
			child.PaintIfNeeded(cb)
		}
	}
}

func (c *Container) NeedPaint() {
	c.needPaint = true

	// bubble flag up in parents
	for c2 := c.Parent; c2 != nil; c2 = c2.Parent {
		c2.childNeedsPaint = true
	}
}
