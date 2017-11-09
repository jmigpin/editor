package widget

import (
	"image"
	"image/color"
	"math"

	"github.com/jmigpin/editor/imageutil"
	"github.com/jmigpin/editor/uiutil/event"
)

type ScrollArea struct {
	ContainerEmbedNode
	ScrollWidth int
	LeftScroll  bool
	VBar        ScrollBar

	content Node
	updater ScrollAreaUpdater
}

func (sa *ScrollArea) Init(ctx Context, updater ScrollAreaUpdater, content Node) {
	*sa = ScrollArea{
		ScrollWidth: 10,
		LeftScroll:  true,
		content:     content,
		updater:     updater,
	}
	sa.SetWrapper(sa)
	sa.VBar.Init(ctx, sa)
	sa.Append(&sa.VBar, sa.content)
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

func (sa *ScrollArea) calcPositionFromPoint(p *image.Point) {
	// Called when dragging the scrollbar

	py := float64(p.Sub(sa.VBar.pressPad).Sub(sa.VBar.Bounds().Min).Y)
	dy := float64(sa.VBar.Bounds().Dy())

	offset := py / dy
	height := 1.0
	viewh := sa.VBar.sizePercent

	// avoid small adjustments to the textarea if the handle doesn't move

	sa.CalcPosition(offset, height, viewh)
	sa.updater.UpdatePositionFromPoint()
}

func (sa *ScrollArea) PageUp()   { sa.scrollPage(true) }
func (sa *ScrollArea) PageDown() { sa.scrollPage(false) }
func (sa *ScrollArea) scrollPage(up bool) {
	// TODO: real scroll size to allow accurate pageup/down on big files

	// page up/down through the scrollbar handle size
	vb := sa.VBar.Handle.Bounds()
	hsize := vb.Dy()
	mult := 0.95
	if up {
		mult *= -1
	}
	y := vb.Min.Y + int(float64(hsize)*mult)
	sa.calcPositionFromPoint(&image.Point{0, y})
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
				sa.updater.CalcPositionFromScroll(true)
			case evt.Button == event.ButtonWheelDown:
				sa.updater.CalcPositionFromScroll(false)
			}
		}
	}
	return false
}

type ScrollAreaUpdater interface {
	UpdatePositionFromPoint()
	CalcPositionFromScroll(bool)
}

type ScrollBar struct {
	ShellEmbedNode
	Handle ScrollHandle
	Color  *color.Color

	sizePercent     float64
	positionPercent float64
	pressPad        image.Point
	sa              *ScrollArea

	clicking bool
	dragging bool

	ctx Context
}

func (sb *ScrollBar) Init(ctx Context, sa *ScrollArea) {
	*sb = ScrollBar{ctx: ctx, sa: sa}
	sb.SetWrapper(sb)
	sb.positionPercent = 0.0
	sb.sizePercent = 1.0

	sb.Handle.Init(ctx, sb)
	sb.Handle.SetNotDraggable(true)
	sb.Append(&sb.Handle)
}
func (sb *ScrollBar) setPressPad(p *image.Point) {
	b := sb.Handle.Bounds()
	if p.In(b) {
		// set position relative to the bar top
		sb.pressPad.X = p.X - b.Min.X
		sb.pressPad.Y = p.Y - b.Min.Y
	} else {
		// set position in the middle of the bar
		sb.pressPad.X = b.Dx() / 2
		sb.pressPad.Y = b.Dy() / 2
	}
}
func (sb *ScrollBar) Measure(hint image.Point) image.Point {
	return image.Point{}
}
func (sb *ScrollBar) Paint() {
	if sb.Color == nil {
		return
	}
	u := sb.Bounds()
	imageutil.FillRectangle(sb.ctx.Image(), &u, *sb.Color)
}
func (sb *ScrollBar) OnInputEvent(ev interface{}, p image.Point) bool {
	switch evt := ev.(type) {
	case *event.MouseDown:
		switch evt.Button {
		case event.ButtonLeft:
			sb.clicking = true
			sb.setPressPad(&evt.Point)
			sb.sa.calcPositionFromPoint(&evt.Point)
		case event.ButtonWheelUp:
			sb.sa.PageUp()
		case event.ButtonWheelDown:
			sb.sa.PageDown()
		}
	case *event.MouseMove:
		if sb.clicking {
			sb.sa.calcPositionFromPoint(&evt.Point)
		}
	case *event.MouseUp:
		if sb.clicking {
			sb.clicking = false
			sb.sa.calcPositionFromPoint(&evt.Point)
			sb.MarkNeedsPaint()
		}

	case *event.MouseDragStart:
		sb.clicking = false
		sb.dragging = true
		sb.sa.calcPositionFromPoint(&evt.Point)
	case *event.MouseDragMove:
		sb.sa.calcPositionFromPoint(&evt.Point)
	case *event.MouseDragEnd:
		sb.dragging = false
		sb.sa.calcPositionFromPoint(&evt.Point)
	}
	return false
}

type ScrollHandle struct {
	LeafEmbedNode
	Color *color.Color

	ctx    Context
	sb     *ScrollBar
	inside bool
}

func (sh *ScrollHandle) Init(ctx Context, sb *ScrollBar) {
	*sh = ScrollHandle{ctx: ctx, sb: sb}
	sh.SetWrapper(sh)
}
func (sh *ScrollHandle) Measure(hint image.Point) image.Point {
	return image.Point{}
}
func (sh *ScrollHandle) Paint() {
	if sh.Color == nil {
		return
	}
	var c color.Color
	if sh.sb.clicking || sh.sb.dragging {
		c = *sh.Color
	} else if sh.inside {
		c = imageutil.Tint(*sh.Color, 0.30)
	} else {
		// normal
		c = imageutil.Tint(*sh.Color, 0.40)
	}
	b := sh.Bounds()
	imageutil.FillRectangle(sh.ctx.Image(), &b, c)
}
func (sh *ScrollHandle) OnInputEvent(ev interface{}, p image.Point) bool {
	switch ev.(type) {
	case *event.MouseEnter:
		sh.inside = true
		sh.MarkNeedsPaint()
	case *event.MouseLeave:
		sh.inside = false
		sh.MarkNeedsPaint()
	}
	return false
}
