package widget

import (
	"image"
	"math"

	"github.com/jmigpin/editor/util/imageutil"
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
	o := sb.positionPercent
	if sb.Horizontal {
		px := float64(p.Sub(sb.pressPad).Sub(sb.Bounds.Min).X)
		o = px / float64(sb.Bounds.Dx())
	} else {
		py := float64(p.Sub(sb.pressPad).Sub(sb.Bounds.Min).Y)
		o = py / float64(sb.Bounds.Dy())
	}
	sb.scrollOffsetPercent(o)
}

//----------

func (sb *ScrollBar) scrollPage(up bool) {
	size := sb.sa.scrollable.ScrollableSize()
	marginv := sb.sa.scrollable.ScrollablePagingMargin()
	margin := float64(marginv) / float64(size.Y) // TODO
	v := sb.sizePercent - margin

	// deal with small spaces
	if v < 0 {
		v = 0.001
	}

	sb.scrollAmount(v, up)
}

func (sb *ScrollBar) scrollJump(up bool) {
	size := sb.sa.scrollable.ScrollableSize()
	jumpv := sb.sa.scrollable.ScrollableScrollJump()
	jump := float64(jumpv) / float64(size.Y) // TODO
	v := jump

	// deal with small spaces
	const j = 4
	if sb.sizePercent < jump*j {
		v = sb.sizePercent / j
		if v < 0 {
			v = 0.001
		}
	}

	sb.scrollAmount(v, up)
}

func (sb *ScrollBar) scrollAmount(amountPerc float64, up bool) {
	if up {
		amountPerc = -amountPerc
	}
	o := sb.positionPercent + amountPerc
	sb.scrollOffsetPercent(o)
}

//----------

func (sb *ScrollBar) scrollOffsetPercent(offsetPerc float64) {
	size := 1.0
	vsize := sb.sizePercent
	sb.calcPositionAndSize(size, vsize, offsetPerc)

	// scroll scrollable
	{
		size := sb.sa.scrollable.ScrollableSize()
		offset := sb.sa.scrollable.ScrollableOffset()
		if sb.Horizontal {
			offset.X = int(sb.positionPercent * float64(size.X))
		} else {
			offset.Y = int(sb.positionPercent * float64(size.Y))
		}
		sb.sa.scrollable.SetScrollableOffset(offset)
	}
}

//----------

func (sb *ScrollBar) calcPositionAndSize(size, viewSize, offset float64) {
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

	d := bsize.Y
	if sb.Horizontal {
		d = bsize.X
	}

	size := sb.sa.scrollable.ScrollableSize()
	vsize := sb.sa.scrollable.Embed().Bounds.Size() // view size
	offset := sb.sa.scrollable.ScrollableOffset()
	if sb.Horizontal {
		sb.calcPositionAndSize(float64(size.X), float64(vsize.X), float64(offset.X))
	} else {
		sb.calcPositionAndSize(float64(size.Y), float64(vsize.Y), float64(offset.Y))
	}

	pp := int(math.Ceil(float64(d) * sb.positionPercent))
	sp := int(math.Ceil(float64(d) * sb.sizePercent))

	// minimum bar size (stay visible)
	minSize := 4
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
