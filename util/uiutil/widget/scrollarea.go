package widget

import (
	"image"
	"math"

	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/uiutil/event"
)

type ScrollArea struct {
	EmbedNode
	ScrollWidth int
	LeftScroll  bool
	VSBar       *ScrollBar
	HSBar       *ScrollBar

	scrollable Scrollable
}

func NewScrollArea(ctx ImageContext, scrollable Scrollable, vert, horiz bool) *ScrollArea {
	sa := &ScrollArea{
		ScrollWidth: 10,
		LeftScroll:  true,
		scrollable:  scrollable,
	}
	if vert {
		sa.VSBar = NewScrollBar(ctx, sa)
		sa.Append(sa.VSBar)
	}
	if horiz {
		sa.HSBar = NewScrollBar(ctx, sa)
		sa.HSBar.Horizontal = true
		sa.Append(sa.HSBar)
	}
	sa.Append(sa.scrollable)
	sa.scrollable.SetScroller(sa)
	return sa
}

func (sa *ScrollArea) scrollPageUp()   { sa.scrollPage(true) }
func (sa *ScrollArea) scrollPageDown() { sa.scrollPage(false) }
func (sa *ScrollArea) scrollPage(up bool) {
	if sa.VSBar != nil {
		size := sa.scrollable.ScrollableSize()
		marginv := sa.scrollable.ScrollablePagingMargin()
		margin := float64(marginv) / float64(size.Y)
		v := sa.VSBar.sizePercent - margin

		// deal with small spaces
		if v < 0 {
			v = 0.001
		}

		sa.scrollAmount(v, up)
	}
}

func (sa *ScrollArea) scrollUp()   { sa.scroll(true) }
func (sa *ScrollArea) scrollDown() { sa.scroll(false) }
func (sa *ScrollArea) scroll(up bool) {
	if sa.VSBar != nil {
		size := sa.scrollable.ScrollableSize()
		jumpv := sa.scrollable.ScrollableScrollJump()
		jump := float64(jumpv) / float64(size.Y)
		v := jump

		// deal with small spaces
		const j = 4
		if sa.VSBar.sizePercent < jump*j {
			v = sa.VSBar.sizePercent / j
			if v < 0 {
				v = 0.001
			}
		}

		sa.scrollAmount(v, up)
	}
}

func (sa *ScrollArea) scrollAmount(amount float64, up bool) {
	if up {
		amount = -amount
	}
	yo := sa.VSBar.positionPercent + amount
	sa.scrollOffset(0, yo)
}

func (sa *ScrollArea) scrollToPoint(p *image.Point, horiz bool) {
	var xo, yo float64
	if sa.VSBar != nil {
		yo = sa.VSBar.positionPercent
		if !horiz {
			py := float64(p.Sub(sa.VSBar.pressPad).Sub(sa.VSBar.Bounds.Min).Y)
			yo = py / float64(sa.VSBar.Bounds.Dy())
		}
	}
	if sa.HSBar != nil {
		xo = sa.HSBar.positionPercent
		if horiz {
			px := float64(p.Sub(sa.HSBar.pressPad).Sub(sa.HSBar.Bounds.Min).X)
			xo = px / float64(sa.HSBar.Bounds.Dx())
		}
	}
	sa.scrollOffset(xo, yo)
}

func (sa *ScrollArea) scrollOffset(xo, yo float64) {
	size := sa.scrollable.ScrollableSize()
	s := 1.0
	var xoo, yoo int
	if sa.VSBar != nil {
		vs := sa.VSBar.sizePercent
		sa.VSBar.CalcSizePosition(s, vs, yo)
		sa.VSBar.CalcChildsBounds()
		sa.VSBar.MarkNeedsPaint() // TODO: check if really needs paint
		yoo = int(sa.VSBar.positionPercent * float64(size.Y))
	}
	if sa.HSBar != nil {
		vs := sa.HSBar.sizePercent
		sa.HSBar.CalcSizePosition(s, vs, xo)
		sa.HSBar.CalcChildsBounds()
		sa.HSBar.MarkNeedsPaint() // TODO: check if really needs paint
		xoo = int(sa.HSBar.positionPercent * float64(size.X))
	}
	sa.scrollable.SetScrollableOffset(image.Point{xoo, yoo})
}

// Implement Scroller interface.
func (sa *ScrollArea) SetScrollerOffset(offset image.Point) {
	size := sa.scrollable.ScrollableSize()
	vsize := sa.scrollable.Embed().Bounds.Size()
	if sa.VSBar != nil {
		s := size.Y
		vs := vsize.Y
		o := offset.Y
		sa.VSBar.CalcSizePosition(float64(s), float64(vs), float64(o))
		sa.VSBar.CalcChildsBounds()
		sa.VSBar.MarkNeedsPaint() // TODO: check if really needs paint
	}
	if sa.HSBar != nil {
		s := size.X
		vs := vsize.X
		o := offset.X
		sa.HSBar.CalcSizePosition(float64(s), float64(vs), float64(o))
		sa.HSBar.CalcChildsBounds()
		sa.HSBar.MarkNeedsPaint() // TODO: check if really needs paint
	}
}

func (sa *ScrollArea) Measure(hint image.Point) image.Point {
	h := hint
	h.X -= sa.ScrollWidth
	h = imageutil.MaxPoint(h, image.Point{0, 0})
	m := sa.EmbedNode.Measure(h)
	m.X += sa.ScrollWidth
	m = imageutil.MinPoint(m, hint)
	return m
}

func (sa *ScrollArea) CalcChildsBounds() {
	b := sa.Bounds
	if sa.VSBar != nil {
		r := b
		if sa.LeftScroll {
			r.Max.X = r.Min.X + sa.ScrollWidth
			b.Min.X = r.Max.X
		} else {
			r.Min.X = r.Max.X - sa.ScrollWidth
			b.Max.X = r.Min.X
		}
		sa.VSBar.Bounds = r
		sa.VSBar.CalcChildsBounds()
	}
	if sa.HSBar != nil {
		r := b
		r.Min.Y = r.Max.Y - sa.ScrollWidth
		b.Max.Y = r.Min.Y
		sa.HSBar.Bounds = r
		sa.HSBar.CalcChildsBounds()
	}
	// scrollable bounds
	{
		sa.scrollable.Embed().Bounds = b.Intersect(sa.Bounds)
		sa.scrollable.CalcChildsBounds()
	}
}

func (sa *ScrollArea) OnInputEvent(ev0 interface{}, p image.Point) bool {
	switch evt := ev0.(type) {
	case *event.KeyDown:
		switch evt.Code {
		case event.KCodePageUp:
			sa.scrollPageUp()
		case event.KCodePageDown:
			sa.scrollPageDown()
		}
	case *event.MouseDown:
		// scrolling with the wheel on the content area
		if p.In(sa.scrollable.Embed().Bounds) {
			switch {
			case evt.Button == event.ButtonWheelUp:
				sa.scrollUp()
			case evt.Button == event.ButtonWheelDown:
				sa.scrollDown()
			}
		}
	}
	return false
}

// Parent of the scroll handle.
type ScrollBar struct {
	EmbedNode
	Handle     *ScrollHandle
	Horizontal bool

	sizePercent     float64
	positionPercent float64
	pressPad        image.Point
	sa              *ScrollArea

	clicking bool
	dragging bool

	ctx ImageContext
}

func NewScrollBar(ctx ImageContext, sa *ScrollArea) *ScrollBar {
	sb := &ScrollBar{ctx: ctx, sa: sa}
	sb.positionPercent = 0.0
	sb.sizePercent = 1.0

	sb.Handle = NewScrollHandle(ctx, sb)
	sb.Handle.SetNotDraggable(true)
	sb.Append(sb.Handle)
	return sb
}

func (sb *ScrollBar) CalcSizePosition(size, viewSize, offset float64) {
	pp := 0.0
	sp := 1.0
	if size > viewSize {
		dh := size - viewSize
		if offset < 0 {
			offset = 0
		} else if offset > dh {
			offset = dh
		}
		pp = offset / size
		sp = viewSize / size
		if sp > 1 {
			sp = 1
		}
	}
	sb.sizePercent = sp
	sb.positionPercent = pp
}

func (sb *ScrollBar) Measure(hint image.Point) image.Point {
	return image.Point{}
}

func (sb *ScrollBar) CalcChildsBounds() {
	bsize := sb.Bounds.Size()
	r := sb.Bounds

	d := bsize.Y
	if sb.Horizontal {
		d = bsize.X
	}

	pp := int(math.Ceil(float64(d) * sb.positionPercent))
	sp := int(math.Ceil(float64(d) * sb.sizePercent))
	minSize := 4 // minimum bar size (stay visible)
	if sp < minSize {
		sp = minSize
	}

	if sb.Horizontal {
		r.Min.X += pp
		r.Max.X = r.Min.X + sp
	} else {
		r.Min.Y += pp
		r.Max.Y = r.Min.Y + sp
	}
	r = r.Intersect(sb.Bounds)

	sb.Handle.Bounds = r
	sb.Handle.CalcChildsBounds()
}

func (sb *ScrollBar) setPressPad(p *image.Point) {
	b := sb.Handle.Bounds
	if p.In(b) {
		// set position relative to the bar top-left
		sb.pressPad.X = p.X - b.Min.X
		sb.pressPad.Y = p.Y - b.Min.Y
	} else {
		// set position in the middle of the bar
		sb.pressPad.X = b.Dx() / 2
		sb.pressPad.Y = b.Dy() / 2
	}
}

func (sb *ScrollBar) Paint() {
	c := sb.Theme.Palette().Normal.Bg
	imageutil.FillRectangle(sb.ctx.Image(), &sb.Bounds, c)
}

func (sb *ScrollBar) OnInputEvent(ev interface{}, p image.Point) bool {
	switch evt := ev.(type) {
	case *event.MouseDown:
		switch evt.Button {
		case event.ButtonLeft:
			sb.clicking = true
			sb.setPressPad(&evt.Point)
			sb.sa.scrollToPoint(&evt.Point, sb.Horizontal)
		case event.ButtonWheelUp:
			sb.sa.scrollPageUp()
		case event.ButtonWheelDown:
			sb.sa.scrollPageDown()
		}
	case *event.MouseMove:
		if sb.clicking {
			sb.sa.scrollToPoint(&evt.Point, sb.Horizontal)
		}
	case *event.MouseUp:
		if sb.clicking {
			sb.clicking = false
			sb.sa.scrollToPoint(&evt.Point, sb.Horizontal)
		}

	case *event.MouseDragStart:
		sb.clicking = false
		sb.dragging = true
		sb.sa.scrollToPoint(&evt.Point, sb.Horizontal)
	case *event.MouseDragMove:
		sb.sa.scrollToPoint(&evt.Point, sb.Horizontal)
	case *event.MouseDragEnd:
		sb.dragging = false
		sb.sa.scrollToPoint(&evt.Point, sb.Horizontal)
	}
	return false
}

type ScrollHandle struct {
	EmbedNode
	ctx    ImageContext
	sb     *ScrollBar
	inside bool
}

func NewScrollHandle(ctx ImageContext, sb *ScrollBar) *ScrollHandle {
	sh := &ScrollHandle{ctx: ctx, sb: sb}
	return sh
}
func (sh *ScrollHandle) Measure(hint image.Point) image.Point {
	return image.Point{}
}
func (sh *ScrollHandle) Paint() {
	c := sh.Theme.Palette().Normal.Fg
	if sh.sb.clicking || sh.sb.dragging {
		c = sh.Theme.Palette().Selection.Fg
	} else if sh.inside {
		c = sh.Theme.Palette().Highlight.Fg
	}
	imageutil.FillRectangle(sh.ctx.Image(), &sh.Bounds, c)
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
