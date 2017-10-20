package widget

import (
	"image"
	"image/color"
	"math"

	"github.com/jmigpin/editor/imageutil"
	"github.com/jmigpin/editor/uiutil/event"
	"golang.org/x/image/colornames"
)

type ScrollArea struct {
	EmbedNode
	ScrollWidth int
	LeftScroll  bool
	VBar        ScrollBar

	ui      UIer
	wrapper ScrollAreaWrapper
	content Node
}

func (sa *ScrollArea) Init(wrapper ScrollAreaWrapper, ui UIer, content Node) {
	*sa = ScrollArea{
		ui:          ui,
		ScrollWidth: 10,
		LeftScroll:  true,
	}
	sa.VBar.Init(ui, sa)
	sa.wrapper = wrapper
	sa.content = content
	AppendChilds(wrapper, &sa.VBar, content)

	// sanity check
	if !sa.HasChild(content) {
		panic("wrapper has different element")
	}
}

func (sa *ScrollArea) CalcPosition(offset, height, viewh float64) {
	pp := 0.0
	sp := 1.0
	if height > viewh {
		dh := height - viewh
		if offset < 0 {
			offset = 0
		} else if offset > dh {
			offset = dh
		}
		pp = offset / height
		sp = viewh / height
		if sp > 1 {
			sp = 1
		}
	}
	sa.VBar.sizePercent = sp
	sa.VBar.positionPercent = pp
}

func (sa *ScrollArea) CalcPositionFromPoint(p *image.Point) {
	// Called when dragging the scrollbar

	py := float64(p.Sub(sa.VBar.pressPad).Sub(sa.VBar.Bounds().Min).Y)
	dy := float64(sa.VBar.Bounds().Dy())

	offset := py / dy
	height := 1.0
	viewh := sa.VBar.sizePercent

	sa.CalcPosition(offset, height, viewh)

	// call wrapper update
	sa.wrapper.UpdatePositionFromPoint()
}

func (sa *ScrollArea) PageUp()   { sa.scrollPage(true) }
func (sa *ScrollArea) PageDown() { sa.scrollPage(false) }
func (sa *ScrollArea) scrollPage(up bool) {
	// TODO: real scroll size to allow accuratepageup/down on big files

	// page up/down through the scrollbar handle size
	vb := sa.VBar.Handle.Bounds()
	hsize := vb.Dy()
	mult := 0.95
	if up {
		mult *= -1
	}
	y := vb.Min.Y + int(float64(hsize)*mult)
	sa.CalcPositionFromPoint(&image.Point{0, y})
}

func (sa *ScrollArea) SetVBarPressPad(p *image.Point) {
	b := sa.VBar.Handle.Bounds()
	if p.In(b) {
		// set position relative to the bar top
		sa.VBar.pressPad.X = p.X - b.Min.X
		sa.VBar.pressPad.Y = p.Y - b.Min.Y
	} else {
		// set position in the middle of the bar
		sa.VBar.pressPad.X = b.Dx() / 2
		sa.VBar.pressPad.Y = b.Dy() / 2
	}
}

func (sa *ScrollArea) VBarPositionPercent() float64 {
	return sa.VBar.positionPercent
}

func (sa *ScrollArea) Measure(hint image.Point) image.Point {
	// Not measuring child or a big value could be passed up.
	// A scrollarea allows the child node to be small.

	return image.Point{50, 50}
}

func (sa *ScrollArea) CalcChildsBounds() {
	if len(sa.Childs()) == 0 {
		return
	}

	// bar
	sa.VBar.bounds = sa.Bounds()
	vbb := &sa.VBar.bounds
	if sa.LeftScroll {
		vbb.Max.X = vbb.Min.X + sa.ScrollWidth
	} else {
		vbb.Min.X = vbb.Max.X - sa.ScrollWidth
	}

	// scroll
	r2 := *vbb
	r2.Min.Y += int(math.Ceil(float64(vbb.Dy()) * sa.VBar.positionPercent))
	size := int(math.Ceil(float64(vbb.Dy()) * sa.VBar.sizePercent))
	if size < 3 {
		size = 3 // minimum bar size (stay visible)
	}
	r2.Max.Y = r2.Min.Y + size
	r2 = r2.Intersect(*vbb)
	sa.VBar.Handle.SetBounds(&r2)

	// child bounds
	r := sa.Bounds()
	if sa.LeftScroll {
		r.Min.X = sa.VBar.bounds.Max.X
	} else {
		r.Max.X = sa.VBar.bounds.Min.X
	}
	sa.content.SetBounds(&r)
	sa.content.CalcChildsBounds()
}

func (sa *ScrollArea) OnInputEvent(ev0 interface{}, p image.Point) bool {
	switch evt := ev0.(type) {
	case *event.KeyDown:
		switch evt.Code {
		case event.KCodePageUp:
			sa.PageUp()
		case event.KCodePageDown:
			sa.PageDown()
		}
	case *event.MouseDown:
		// line scrolling with the wheel on the content area
		if p.In(sa.content.Bounds()) {
			switch {
			case evt.Button == event.ButtonWheelUp:
				sa.wrapper.CalcPositionFromScroll(true)
			case evt.Button == event.ButtonWheelDown:
				sa.wrapper.CalcPositionFromScroll(false)
			}
		}
	}
	return false
}

type ScrollAreaWrapper interface {
	Node
	UpdatePositionFromPoint()
	CalcPositionFromScroll(bool)
}

type ScrollBar struct {
	ShellEmbedNode
	Color  color.Color
	Handle ScrollHandle

	ui              UIer
	sizePercent     float64
	positionPercent float64
	pressPad        image.Point
	sa              *ScrollArea
}

func (sb *ScrollBar) Init(ui UIer, sa *ScrollArea) {
	sb.ui = ui
	sb.sa = sa
	sb.positionPercent = 0.0
	sb.sizePercent = 1.0
	sb.Color = colornames.Antiquewhite

	sb.Handle.ui = ui
	sb.Handle.sa = sa
	sb.Handle.Color = colornames.Orange
	AppendChilds(sb, &sb.Handle)
}
func (sb *ScrollBar) Measure(hint image.Point) image.Point {
	return image.Point{}
}
func (sb *ScrollBar) Paint() {
	u := sb.Bounds()
	sb.ui.FillRectangle(&u, sb.Color)
}
func (sb *ScrollBar) OnInputEvent(ev interface{}, p image.Point) bool {
	switch evt := ev.(type) {
	case *event.MouseDown:
		switch evt.Button {
		case event.ButtonLeft:
			sb.Handle.clickdrag = true
			sb.sa.SetVBarPressPad(&evt.Point)
			sb.sa.CalcPositionFromPoint(&evt.Point)

			// TODO: disabled to avoid other row events when scrolling the row square that calculates new nodes positions
			//case ev.Button == event.ButtonWheelUp:
			//	sa.PageUp()
			//case ev.Button == event.ButtonWheelDown:
			//	sa.PageDown()
		}

	case *event.MouseUp:
		sb.Handle.clickdrag = false
		sb.MarkNeedsPaint()

	case *event.MouseDragStart:
		sb.Handle.clickdrag = true
		sb.sa.CalcPositionFromPoint(&evt.Point)
	case *event.MouseDragMove:
		sb.sa.CalcPositionFromPoint(&evt.Point)
	case *event.MouseDragEnd:
		sb.Handle.clickdrag = false
		sb.sa.CalcPositionFromPoint(&evt.Point)
	}
	return false
}

type ScrollHandle struct {
	LeafEmbedNode
	Color color.Color

	ui        UIer
	inside    bool
	clickdrag bool

	sa *ScrollArea
}

func (sh *ScrollHandle) Measure(hint image.Point) image.Point {
	return image.Point{}
}
func (sh *ScrollHandle) Paint() {
	var c color.Color
	if sh.clickdrag {
		c = sh.Color
	} else if sh.inside {
		c = imageutil.Tint(sh.Color, 0.30)
	} else {
		// normal
		c = imageutil.Tint(sh.Color, 0.40)
	}
	b := sh.Bounds()
	sh.ui.FillRectangle(&b, c)
}
func (sh *ScrollHandle) OnInputEvent(ev interface{}, p image.Point) bool {
	switch evt := ev.(type) {
	case *event.MouseEnter:
		sh.inside = true
		sh.MarkNeedsPaint()
	case *event.MouseLeave:
		sh.inside = false
		sh.MarkNeedsPaint()

	case *event.MouseDown:
		sh.clickdrag = true
		sh.sa.SetVBarPressPad(&evt.Point)
		sh.sa.CalcPositionFromPoint(&evt.Point)
	case *event.MouseUp:
		sh.clickdrag = false
		sh.MarkNeedsPaint()

	case *event.MouseDragStart:
		sh.clickdrag = true
		sh.sa.CalcPositionFromPoint(&evt.Point)
	case *event.MouseDragMove:
		sh.sa.CalcPositionFromPoint(&evt.Point)
	case *event.MouseDragEnd:
		sh.clickdrag = false
		sh.sa.CalcPositionFromPoint(&evt.Point)
	}
	return false
}
