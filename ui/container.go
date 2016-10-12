package ui

import (
	"image"
	"image/color"

	"github.com/jmigpin/editor/drawutil"
)

type Container struct {
	Area   image.Rectangle
	childs []*Container
	UI     *UI

	Painter   Painter // paints this container
	needPaint bool

	OnPointEvent func(*image.Point, Event) bool
}

type Painter interface {
	CalcArea(area *image.Rectangle)
	Paint()
}

func (c *Container) FillRectangle(rect *image.Rectangle, col color.Color) {
	drawutil.FillRectangle(c.UI.RootImage(), rect, col)
}

func (c *Container) AddChilds(cs ...*Container) {
	for _, child := range cs {
		c.childs = append(c.childs, child)
		child.propagateUI(c.UI)
	}
}
func (c *Container) propagateUI(ui *UI) {
	if ui == nil {
		return
	}
	c.UI = ui
	for _, child := range c.childs {
		child.propagateUI(ui)
	}
}
func (c *Container) RemoveChild(c2 *Container) {
	for i, child := range c.childs {
		if child == c2 {
			// remove: ensure garbage collection
			var u []*Container
			u = append(u, c.childs[:i]...)
			u = append(u, c.childs[i+1:]...)
			c.childs = u
			return
		}
	}
}

func (c *Container) TreePaint() {
	if c.needPaint {
		c.ClearNeedPaint()
		//c.Painter.CalcArea(&c.Area)
		c.Painter.Paint()
		c.UI.SendRootImage(&c.Area)
	} else {
		for _, c2 := range c.childs {
			c2.TreePaint()
		}
	}
}
func (c *Container) NeedPaint() {
	c.needPaint = true
}
func (c *Container) ClearNeedPaint() {
	c.needPaint = false
	for _, c2 := range c.childs {
		c2.ClearNeedPaint()
	}
}
func (c *Container) CalcOwnArea() {
	c.Painter.CalcArea(&c.Area)
}

func (c *Container) pointEvent(p *image.Point, ev Event) bool {
	if !p.In(c.Area) {
		if c.Painter == c.UI.Layout {
			// special case: layout accepts events outside (all)
		} else {
			return true
		}
	}
	// first handle the parent, with chance of canceling children
	if c.OnPointEvent != nil {
		ok := c.OnPointEvent(p, ev)
		if !ok {
			return false
		}
	}
	// handle children
	for _, child := range c.childs {
		ok := child.pointEvent(p, ev)
		if !ok {
			return false
		}
	}
	return true
}
