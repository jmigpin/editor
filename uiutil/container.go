package uiutil

import (
	"image"
	"sync"
)

type Container struct {
	Owner interface{}

	Bounds image.Rectangle

	Parent, PrevSibling, NextSibling, FirstChild, LastChild *Container

	NChilds int

	needMeasure     bool
	needPaint       bool
	childNeedsPaint bool

	PaintFunc  func()
	OnCalcFunc func()

	Style Style
}

func (c *Container) InsertChildBefore(cc, next *Container) {
	if cc.Parent != nil && cc.NextSibling != nil || cc.PrevSibling != nil {
		panic("container is already attached")
	}
	if next != nil && next.Parent != c {
		panic("not a child of this container")
	}

	c.NChilds++
	cc.Parent = c

	var prev *Container
	if next == nil {
		if c.LastChild != nil {
			prev = c.LastChild
		}
	} else {
		prev = next.PrevSibling
	}

	connectContainers(prev, cc)
	connectContainers(cc, next)

	if c.FirstChild == nil || c.FirstChild == next {
		c.FirstChild = cc
	}
	if c.LastChild == nil || next == nil {
		c.LastChild = cc
	}
}
func (c *Container) RemoveChild(cc *Container) {
	if cc.Parent != c {
		panic("not a child of this container")
	}

	connectContainers(cc.PrevSibling, cc.NextSibling)

	if c.FirstChild == cc {
		c.FirstChild = cc.NextSibling
	}
	if c.LastChild == cc {
		c.LastChild = cc.PrevSibling
	}

	c.NChilds--
	cc.Parent = nil
	cc.PrevSibling = nil
	cc.NextSibling = nil
}

func connectContainers(a, b *Container) {
	if a != nil {
		a.NextSibling = b
	}
	if b != nil {
		b.PrevSibling = a
	}
}

func (c *Container) AppendChilds(cs ...*Container) {
	for _, c2 := range cs {
		c.InsertChildBefore(c2, nil)
	}
}

func (c *Container) Childs() []*Container {
	var u []*Container
	for h := c.FirstChild; h != nil; h = h.NextSibling {
		u = append(u, h)
	}
	return u
}

func (c *Container) IsAPrevSiblingOf(c2 *Container) bool {
	for u := c2.PrevSibling; u != nil; u = u.PrevSibling {
		if u == c {
			return true
		}
	}
	return false
}
func (c *Container) IsANextSiblingOf(c2 *Container) bool {
	for u := c2.NextSibling; u != nil; u = u.NextSibling {
		if u == c {
			return true
		}
	}
	return false
}

func (c *Container) SwapWithSibling(c2 *Container) {
	if c.Parent != c2.Parent {
		panic("containers don't have the same parent")
	}

	a1, b1 := c.PrevSibling, c.NextSibling
	a2, b2 := c2.PrevSibling, c2.NextSibling

	if a1 == c2 {
		a1 = c
	}
	if b1 == c2 {
		b1 = c
	}
	if a2 == c {
		a2 = c2
	}
	if b2 == c {
		b2 = c2
	}

	connectContainers(a1, c2)
	connectContainers(c2, b1)
	connectContainers(a2, c)
	connectContainers(c, b2)

	p := c.Parent
	if p.FirstChild == c {
		p.FirstChild = c2
	} else if p.FirstChild == c2 {
		p.FirstChild = c
	}
	if p.LastChild == c {
		p.LastChild = c2
	} else if p.LastChild == c2 {
		p.LastChild = c
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
	for child := c.FirstChild; child != nil; child = child.NextSibling {
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
		for child := c.FirstChild; child != nil; child = child.NextSibling {
			child.PaintIfNeeded(cb)
		}
	}
}

func (c *Container) NeedPaint() {
	c.needPaint = true
	// set flag in parents
	for c2 := c.Parent; c2 != nil; c2 = c2.Parent {
		c2.childNeedsPaint = true
	}
}
