package widget

import (
	"image"
	"math"

	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/mathutil"
	"github.com/jmigpin/editor/util/uiutil/event"
)

// Used by ScrollArea. Parent of ScrollHandle.
type ScrollBar struct {
	ENode
	Handle     *ScrollHandle
	Horizontal bool

	positionPercent float64
	sizePercent     float64

	pressPad image.Point
	clicking bool
	dragging bool

	sa *ScrollArea

	ctx ImageContext
}

func NewScrollBar(ctx ImageContext, sa *ScrollArea) *ScrollBar {
	sb := &ScrollBar{ctx: ctx, sa: sa}
	sb.positionPercent = 0.0
	sb.sizePercent = 1.0

	sb.Handle = NewScrollHandle(ctx, sb)
	sb.Append(sb.Handle)
	return sb
}

//----------

func (sb *ScrollBar) scrollToPoint(p *image.Point) {
	py := float64(sb.yaxis(p.Sub(sb.pressPad).Sub(sb.Bounds.Min)))
	o := py / float64(sb.yaxis(sb.Bounds.Size()))
	sb.scrollToPositionPercent(o)
}

//----------

func (sb *ScrollBar) scrollPage(up bool) {
	size := sb.sa.scrollable.ScrollableSize()
	marginv := sb.sa.scrollable.ScrollablePagingMargin()
	margin := float64(marginv) / float64(size.Y) // always Y
	v := sb.sizePercent - margin
	v = mathutil.LimitFloat64(v, 0.001, v) // deal with small spaces
	sb.scrollAmount(v, up)
}

func (sb *ScrollBar) scrollJump(up bool) {
	size := sb.sa.scrollable.ScrollableSize()
	jumpv := sb.sa.scrollable.ScrollableScrollJump()
	jump := float64(jumpv) / float64(size.Y) // always Y
	v := jump

	// deal with small spaces
	const j = 4
	if sb.sizePercent < jump*j {
		v = sb.sizePercent / j
		v = mathutil.LimitFloat64(v, 0.001, v)
	}

	sb.scrollAmount(v, up)
}

func (sb *ScrollBar) scrollAmount(amountPerc float64, up bool) {
	if up {
		amountPerc = -amountPerc
	}
	o := sb.positionPercent + amountPerc
	sb.scrollToPositionPercent(o)
}

//----------

func (sb *ScrollBar) scrollToPositionPercent(offsetPerc float64) {
	size := sb.sa.scrollable.ScrollableSize()
	offset := sb.sa.scrollable.ScrollableOffset()
	offsetPerc = mathutil.LimitFloat64(offsetPerc, 0, 1)
	*sb.yaxisPtr(&offset) = int(offsetPerc * float64(sb.yaxis(size)))
	sb.sa.scrollable.SetScrollableOffset(offset)
}

//----------

func (sb *ScrollBar) calcPositionAndSize() {
	pos := sb.sa.scrollable.ScrollableOffset()
	size := sb.sa.scrollable.ScrollableSize()
	vsize := sb.sa.scrollable.ScrollableViewSize()

	posy := float64(sb.yaxis(pos))
	sizey := float64(sb.yaxis(size))
	vsizey := float64(sb.yaxis(vsize))

	sizey = mathutil.LimitFloat64(sizey, 0.0001, sizey)

	pp := posy / sizey
	sp := vsizey / sizey

	sp = mathutil.LimitFloat64(sp, 0, 1)
	pp = mathutil.LimitFloat64(pp, 0, 1)

	sb.sizePercent = sp
	sb.positionPercent = pp
}

//----------

//func (sb *ScrollBar) calcPositionAndSize_(size, viewSize, offset float64) {
//	pp := 0.0
//	sp := 1.0
//	if size > viewSize {
//		dh := size - viewSize
//		if offset < 0 {
//			offset = 0
//		} else if offset > dh {
//			offset = dh
//		}
//		pp = offset / size
//		sp = viewSize / size
//		if sp > 1 {
//			sp = 1
//		}
//	}
//	sb.sizePercent = sp
//	sb.positionPercent = pp
//}

//----------

func (sb *ScrollBar) OnChildMarked(child Node, newMarks Marks) {
	// paint scrollbar background if the handle is getting painted
	if child == sb.Handle {
		if newMarks.HasAny(MarkNeedsPaint) {
			sb.MarkNeedsPaint()
		}
	}
}

//----------

func (sb *ScrollBar) Layout() {
	bsize := sb.Bounds.Size()
	r := sb.Bounds

	d := sb.yaxis(bsize)

	//size := sb.sa.scrollable.ScrollableSize()
	//vsize := sb.sa.scrollable.ScrollableViewSize()
	//offset := sb.sa.scrollable.ScrollableOffset()
	//sy := sb.yaxis(size)
	//vsy := sb.yaxis(vsize)
	//oy := sb.yaxis(offset)
	//sb.calcPositionAndSize(float64(sy), float64(vsy), float64(oy))

	sb.calcPositionAndSize()

	p := int(math.Ceil(float64(d) * sb.positionPercent))
	s := int(math.Ceil(float64(d) * sb.sizePercent))

	s = mathutil.LimitInt(s, 4, s) // minimum bar size (stay visible)

	*sb.yaxisPtr(&r.Min) += p
	*sb.yaxisPtr(&r.Max) = sb.yaxis(r.Min) + s
	r = r.Intersect(sb.Bounds)

	sb.Handle.Bounds = r
}

func (sb *ScrollBar) Paint() {
	c := sb.TreeThemePaletteColor("scrollbar_bg")
	imageutil.FillRectangle(sb.ctx.Image(), &sb.Bounds, c)
}

//----------

func (sb *ScrollBar) OnInputEvent(ev interface{}, p image.Point) event.Handle {
	switch evt := ev.(type) {
	case *event.MouseDown:
		switch evt.Button {
		case event.ButtonLeft:
			sb.clicking = true
			sb.setPressPad(&evt.Point)
			sb.scrollToPoint(&evt.Point)
			sb.MarkNeedsPaint() // in case it didn't move
		case event.ButtonWheelUp:
			sb.scrollPage(true)
		case event.ButtonWheelDown:
			sb.scrollPage(false)
		}
	case *event.MouseMove:
		if sb.clicking {
			sb.scrollToPoint(&evt.Point)
		}
	case *event.MouseUp:
		if sb.clicking {
			sb.clicking = false
			sb.scrollToPoint(&evt.Point)
			sb.MarkNeedsPaint() // in case it didn't move
		}

	case *event.MouseDragStart:
		sb.clicking = false
		sb.dragging = true
		sb.scrollToPoint(&evt.Point)
	case *event.MouseDragMove:
		sb.scrollToPoint(&evt.Point)
	case *event.MouseDragEnd:
		sb.dragging = false
		sb.scrollToPoint(&evt.Point)
		sb.MarkNeedsPaint() // in case it didn't move
	}
	return event.NotHandled
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

//----------

func (sb *ScrollBar) yaxis(p image.Point) int {
	if sb.Horizontal {
		return p.X
	} else {
		return p.Y
	}
}
func (sb *ScrollBar) yaxisPtr(p *image.Point) *int {
	if sb.Horizontal {
		return &p.X
	} else {
		return &p.Y
	}
}
