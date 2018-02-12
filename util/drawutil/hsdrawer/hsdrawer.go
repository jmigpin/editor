package hsdrawer

import (
	"image"
	"image/color"
	"image/draw"
	"log"

	"github.com/jmigpin/editor/util/drawutil/loopers"
	"github.com/jmigpin/editor/util/imageutil"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// TODO: need to check early exit looper
// fixed.Int26_6 integer part ranges from -33554432 to 33554431
//fixedMaxY := fixed.I(33554431).Ceil()

// Highlight and Selection drawer. The empty value is a valid drawer.
type HSDrawer struct {
	Offset image.Point
	Fg     color.Color

	// Args that will affect the cache calculation. These should be set externally and not be used internally other then to compare to cargs.
	Args args

	// args used on the cached data
	cargs args

	// extensions that don't keep state
	CursorIndex          *int
	HighlightWordOpt     *loopers.HighlightWordOpt
	SelectionOpt         *loopers.SelectionOpt
	FlashSelectionOpt    *loopers.FlashSelectionOpt
	HighlightSegmentsOpt *loopers.HighlightSegmentsOpt

	// loopers instances
	strl    loopers.String
	wlinel  loopers.WrapLine
	wlinecl loopers.WrapLineColor
	pdl     loopers.PosData
	dl      loopers.Draw
	bgl     loopers.Bg
	cl      loopers.Colorize
	al      loopers.Annotations
	acl     loopers.AnnotationsColor
	eel     loopers.EarlyExit

	measurement image.Point // full string measurement (no Y bounds)

	// TODO: left/top pad to help cursor looper to be fully draw
	// small pad added to allow the cursor to be fully drawn on first position
	// need to update getindex/getpoint as well
	//pad image.Point
}

// arguments that must match to allow the cached calculations to persist
type args struct {
	Face             font.Face
	Str              string
	FirstLineOffsetX int

	// extensions that keep state and affect cache need to be recalculated
	WrapLineOpt    *loopers.WrapLineOpt
	ColorizeOpt    *loopers.ColorizeOpt
	AnnotationsOpt *loopers.AnnotationsOpt

	maxX int
}

func (d *HSDrawer) initLoopers(img draw.Image, bounds *image.Rectangle) {
	// warnings
	if bounds != nil && d.cargs.maxX != bounds.Dx() {
		log.Printf("hsdrawer: x differ: %v, %v", d.cargs.maxX, bounds.Dx())
	}

	fmaxX := fixed.I(d.cargs.maxX)

	// loopers
	d.strl = loopers.MakeString(d.cargs.Face, d.cargs.Str)
	d.strl.Pen.X = fixed.I(d.cargs.FirstLineOffsetX)
	linel := loopers.MakeLine(&d.strl, fixed.I(d.Offset.X))

	// posdata keepers with fixed index to allow some of them to be unset
	ki := 0
	keepers := [4]loopers.PosDataKeeper{}
	keepers[ki] = &d.strl
	ki++
	if d.cargs.AnnotationsOpt != nil {
		d.al = loopers.MakeAnnotations(&d.strl, &linel, d.cargs.AnnotationsOpt)
		keepers[ki] = &d.al
	}
	ki++
	if d.cargs.WrapLineOpt != nil {
		d.wlinel = loopers.MakeWrapLine(&d.strl, &linel, fmaxX)
		keepers[ki] = &d.wlinel
	}
	ki++
	if d.cargs.ColorizeOpt != nil {
		d.cl = loopers.MakeColorize(&d.strl, d.cargs.ColorizeOpt)
		keepers[ki] = &d.cl
	}
	ki++

	// uses own previous pdl data
	d.pdl = loopers.MakePosData(&d.strl, keepers[:], 250, d.pdl.Data)

	// loopers order
	start := &loopers.EmbedLooper{}
	d.strl.SetOuterLooper(start)
	var outer loopers.Looper = &d.strl
	if d.cargs.AnnotationsOpt != nil {
		d.al.SetOuterLooper(outer)
		outer = &d.al
	}
	linel.SetOuterLooper(outer)
	outer = &linel
	if d.cargs.WrapLineOpt != nil {
		d.wlinel.SetOuterLooper(outer)
		outer = &d.wlinel
	}
	if d.cargs.ColorizeOpt != nil {
		d.cl.SetOuterLooper(outer)
		outer = &d.cl
	}

	// pdl (position data) phase last looper
	d.pdl.SetOuterLooper(outer)

	// draw loopers
	if img == nil || bounds == nil {
		return
	}
	d.dl = loopers.MakeDraw(&d.strl, img, bounds)
	d.bgl = loopers.MakeBg(&d.strl, &d.dl)
	var sl loopers.Selection
	if d.SelectionOpt != nil {
		sl = loopers.MakeSelection(&d.strl, &d.bgl, &d.dl, d.SelectionOpt)
	}
	scl := loopers.MakeSetColors(&d.dl, &d.bgl)
	scl.Fg = d.Fg
	scl.Bg = nil
	var hwl loopers.HighlightWord
	if d.HighlightWordOpt != nil {
		hwl = loopers.MakeHighlightWord(&d.strl, &d.bgl, &d.dl, d.HighlightWordOpt)
	}
	var fsl loopers.FlashSelection
	if d.FlashSelectionOpt != nil {
		fsl = loopers.MakeFlashSelection(&d.strl, &d.bgl, &d.dl, d.FlashSelectionOpt)
	}
	var cursorl loopers.Cursor
	if d.CursorIndex != nil {
		cursorl = loopers.MakeCursor(&d.strl, &d.dl, bounds, *d.CursorIndex)
	}
	if d.cargs.WrapLineOpt != nil {
		d.wlinecl = loopers.MakeWrapLineColor(&d.wlinel, &d.dl, &d.bgl, d.cargs.WrapLineOpt)
	}
	var hsl loopers.HighlightSegments
	if d.HighlightSegmentsOpt != nil {
		hsl = loopers.MakeHighlightSegments(&d.strl, &d.bgl, &d.dl, d.HighlightSegmentsOpt)
	}
	var ccl loopers.ColorizeColor
	if d.cargs.ColorizeOpt != nil {
		ccl = loopers.MakeColorizeColor(&d.dl, &d.cl)
	}
	if d.cargs.AnnotationsOpt != nil {
		d.acl = loopers.MakeAnnotationsColor(&d.al, &d.strl, &d.dl, &d.bgl, d.cargs.AnnotationsOpt)
	}
	d.eel = loopers.MakeEarlyExit(&d.strl, fixed.I(bounds.Size().Y))

	// draw phase (bypasses pdl looper)
	scl.SetOuterLooper(outer)

	outer = &scl
	if d.cargs.ColorizeOpt != nil {
		ccl.SetOuterLooper(outer)
		outer = &ccl
	}
	if d.cargs.AnnotationsOpt != nil {
		d.acl.SetOuterLooper(outer)
		outer = &d.acl
	}
	if d.HighlightWordOpt != nil {
		hwl.SetOuterLooper(outer)
		outer = &hwl
	}
	if d.SelectionOpt != nil {
		sl.SetOuterLooper(outer)
		outer = &sl
	}
	if d.HighlightSegmentsOpt != nil {
		hsl.SetOuterLooper(outer)
		outer = &hsl
	}
	if d.cargs.WrapLineOpt != nil {
		d.wlinecl.SetOuterLooper(outer)
		outer = &d.wlinecl
	}
	if d.FlashSelectionOpt != nil {
		fsl.SetOuterLooper(outer)
		outer = &fsl
	}

	// bg phase last looper
	d.bgl.SetOuterLooper(outer)

	// rune phase (bypasses bgl looper)
	if d.CursorIndex != nil {
		cursorl.SetOuterLooper(outer)
		outer = &cursorl
	}
	d.dl.SetOuterLooper(outer)
}

func (d *HSDrawer) NeedMeasure(maxX int) bool {
	d.Args.maxX = maxX
	return d.Args != d.cargs
}

func (d *HSDrawer) Measure(max image.Point) image.Point {
	d.measure2(max.X, false)
	// bounded measurement, smaller or equal than max
	return imageutil.MinPoint(d.measurement, max)
}

func (d *HSDrawer) measure2(maxX int, fromDraw bool) {
	if !d.NeedMeasure(maxX) {
		return
	}

	if fromDraw {
		log.Printf("hsdrawer: measuring before draw (different args)")
		//d.DebugArgsCompare()
	}

	d.cargs = d.Args

	// reset result
	d.measurement = image.Point{}

	// nothing to do
	if d.cargs.Face == nil {
		return
	}

	// reset cache data
	d.pdl.Data = nil

	d.initLoopers(nil, nil)

	// loopers
	ml := loopers.MakeMeasure(&d.strl)
	// iterator order
	ml.SetOuterLooper(&d.pdl)
	// run measure
	ml.Loop(func() bool { return true })
	m := image.Point{ml.M.X.Ceil(), ml.M.Y.Ceil()}
	d.measurement = m
}

func (d *HSDrawer) Draw(img draw.Image, bounds *image.Rectangle) {
	d.measure2(bounds.Dx(), true)

	// nothing to do
	if d.cargs.Face == nil {
		return
	}

	d.initLoopers(img, bounds)

	// restore position to a close data point (performance)
	p := fixed.P(d.Offset.X, d.Offset.Y)
	d.pdl.RestorePosDataCloseToPoint(&p)
	d.strl.Pen = d.strl.Pen.Sub(p)

	// draw bg first to correctly paint below all runes that are drawn later
	d.eel.SetOuterLooper(&d.bgl)
	d.eel.Loop(func() bool { return true })

	// prepare for draw runes
	d.initLoopers(img, bounds)

	// restore position to a close data point (performance)
	d.pdl.RestorePosDataCloseToPoint(&p)
	d.strl.Pen = d.strl.Pen.Sub(p)

	// draw runes
	d.eel.SetOuterLooper(&d.dl)
	d.eel.Loop(func() bool { return true })
}

func (d *HSDrawer) GetPoint(index int) image.Point {
	if d.pdl.Data == nil {
		return image.Point{}
	}

	d.initLoopers(nil, nil)

	d.pdl.RestorePosDataCloseToIndex(index)
	p := d.pdl.GetPoint(index)                // minimum of pen bounds
	p2 := image.Point{p.X.Ceil(), p.Y.Ceil()} // equal or inside the pen bounds
	return p2
}
func (d *HSDrawer) GetIndex(p *image.Point) int {
	if d.pdl.Data == nil {
		return 0
	}

	d.initLoopers(nil, nil)

	p2 := fixed.P(p.X, p.Y)
	d.pdl.RestorePosDataCloseToPoint(&p2)
	return d.pdl.GetIndex(&p2)
}

func (d *HSDrawer) GetAnnotationsIndex(p *image.Point) (int, int, bool) {
	if d.pdl.Data == nil {
		return 0, 0, false
	}
	if d.Args.AnnotationsOpt == nil {
		return 0, 0, false
	}

	d.initLoopers(nil, nil)

	p2 := fixed.P(p.X, p.Y)
	d.pdl.RestorePosDataCloseToPoint(&p2)

	return loopers.GetAnnotationsIndex(&d.pdl, &d.al, &p2)
}

func (d *HSDrawer) MeasurementFullY() image.Point {
	return d.measurement
}

func (d *HSDrawer) LineHeight() int {
	return d.strl.LineHeight().Ceil()
}

func (d *HSDrawer) DebugArgsCompare() {
	a1 := d.Args
	a2 := d.cargs
	a1.Face = nil
	a2.Face = nil
	//a1.Str = fmt.Sprintf("len=%v", len(a1.Str))
	//a2.Str = fmt.Sprintf("len=%v", len(a2.Str))
	//spew.Dump(a1, a2)
	log.Printf("drawer %p", d)
	log.Printf("Str %v, %v", len(a1.Str), len(a2.Str))
	log.Printf("wrapline %p, %p", a1.WrapLineOpt, a2.WrapLineOpt)
	log.Printf("colorize %p, %p", a1.ColorizeOpt, a2.ColorizeOpt)
	log.Printf("annotations %p, %p", a1.AnnotationsOpt, a2.AnnotationsOpt)
}
