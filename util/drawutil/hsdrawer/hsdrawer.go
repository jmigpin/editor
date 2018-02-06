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

// Highlight and Selection drawer. The empty value is a valid drawer.
type HSDrawer struct {
	Face             font.Face
	Str              string
	Offset           image.Point
	FirstLineOffsetX int
	Fg               color.Color

	// extensions
	CursorIndex          *int
	WrapLineOpt          *loopers.WrapLineOpt
	HighlightWordOpt     *loopers.HighlightWordOpt
	SelectionOpt         *loopers.SelectionOpt
	FlashSelectionOpt    *loopers.FlashSelectionOpt
	HighlightSegmentsOpt *loopers.HighlightSegmentsOpt
	ColorizeOpt          *loopers.ColorizeOpt

	strl    loopers.String
	wlinel  loopers.WrapLine
	pdl     loopers.PosData
	wlinecl loopers.WrapLineColor
	dl      loopers.Draw
	bgl     loopers.Bg
	cl      loopers.Colorize
	eel     loopers.EarlyExit

	measurement image.Point // full string measurement (no Y bounds)

	args args

	// TODO: left/top pad to help cursor looper to be fully draw
	// small pad added to allow the cursor to be fully drawn on first position
	// need to update getindex/getpoint as well
	//pad image.Point
}

// arguments that must match to allow the cached calculations to persist
type args struct {
	face             font.Face
	str              string
	maxX             int
	firstLineOffsetX int
}

func (d *HSDrawer) NeedMeasure(max image.Point) bool {
	a := d.getArgs(max)
	return a != d.args
}

func (d *HSDrawer) getArgs(max image.Point) args {
	return args{
		face:             d.Face,
		str:              d.Str,
		maxX:             max.X,
		firstLineOffsetX: d.FirstLineOffsetX,
	}
}

func (d *HSDrawer) Measure(max image.Point) image.Point {
	if d.Face == nil {
		return image.Point{}
	}

	// use cached value, just need to update the bounded measure
	a := d.getArgs(max)
	if a == d.args {
		// bounded measurement, smaller or equal than max
		return imageutil.MinPoint(d.measurement, max)
	}
	d.args = a

	// TODO: need to check early exit looper
	// fixed.Int26_6 integer part ranges from -33554432 to 33554431
	//fixedMaxY := fixed.I(33554431).Ceil()

	// loopers
	d.pdl.Data = nil // reset data
	d.initMeasurers(d.args.maxX)
	ml := loopers.NewMeasure(&d.strl)

	// iterator order
	ml.SetOuterLooper(&d.pdl)

	// run measure
	ml.Loop(func() bool { return true })
	m := image.Point{ml.M.X.Ceil(), ml.M.Y.Ceil()}
	d.measurement = m

	// bounded measurement, smaller or equal than max
	return imageutil.MinPoint(d.measurement, max)
}

func (d *HSDrawer) MeasurementFullY() image.Point {
	return d.measurement
}

func (d *HSDrawer) Draw(img draw.Image, bounds *image.Rectangle) {
	if d.Face == nil {
		return
	}

	a := d.getArgs(bounds.Size())

	// nothing todo
	if a.maxX == 0 {
		return
	}

	// self check
	if a != d.args {
		log.Printf("hsdrawer: drawing with different args: (maxx=%v vs %v", a.maxX, d.args.maxX)
	}

	// prepare for bg draw
	d.initDrawers(img, bounds)

	// restore position to a close data point (performance)
	p := fixed.P(d.Offset.X, d.Offset.Y)
	d.pdl.RestorePosDataCloseToPoint(&p)
	d.strl.Pen = d.strl.Pen.Sub(p)

	// draw bg first to correctly paint below all runes that are drawn later
	d.eel.SetOuterLooper(&d.bgl)
	d.eel.Loop(func() bool { return true })

	// prepare for draw runes
	d.initDrawers(img, bounds)

	// restore position to a close data point (performance)
	d.pdl.RestorePosDataCloseToPoint(&p)
	d.strl.Pen = d.strl.Pen.Sub(p)

	// draw runes
	d.eel.SetOuterLooper(&d.dl)
	d.eel.Loop(func() bool { return true })
}

func (d *HSDrawer) initMeasurers(maxX int) {
	fmaxX := fixed.I(maxX)

	d.strl = loopers.MakeString(d.Face, d.Str)
	d.strl.Pen.X = fixed.I(d.FirstLineOffsetX)
	linel := loopers.MakeLine(&d.strl, fixed.I(d.Offset.X))

	// TODO: change of keepers length need a d.pdl.data=nil (reset)
	// TODO: ensure keepers from measure?

	keepers := []loopers.PosDataKeeper{&d.strl}
	if d.WrapLineOpt != nil {
		d.wlinel = loopers.MakeWrapLine(&d.strl, &linel, fmaxX)
		keepers = append(keepers, &d.wlinel)
	}
	if d.ColorizeOpt != nil {
		d.cl = loopers.MakeColorize(&d.strl, d.ColorizeOpt)
		keepers = append(keepers, &d.cl)
	}
	d.pdl = loopers.MakePosData(&d.strl, keepers, 250, d.pdl.Data)

	// iterator order
	start := &loopers.EmbedLooper{}
	d.strl.SetOuterLooper(start)
	linel.SetOuterLooper(&d.strl)
	var outer loopers.Looper = &linel
	if d.WrapLineOpt != nil {
		d.wlinel.SetOuterLooper(outer)
		outer = &d.wlinel
	}
	if d.ColorizeOpt != nil {
		d.cl.SetOuterLooper(outer)
		outer = &d.cl
	}
	d.pdl.SetOuterLooper(outer)
}

func (d *HSDrawer) initDrawers(img draw.Image, bounds *image.Rectangle) {
	// TODO: use bounds without the pad for drawing runes, the cursor draws on full bounds
	u := *bounds
	unpaddedBounds := &u

	// loopers
	d.initMeasurers(unpaddedBounds.Size().X)
	d.dl = loopers.MakeDraw(&d.strl, img, unpaddedBounds)
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
	if d.WrapLineOpt != nil {
		d.wlinecl = loopers.MakeWrapLineColor(&d.wlinel, &d.dl, &d.bgl, d.WrapLineOpt)
	}
	var hsl loopers.HighlightSegments
	if d.HighlightSegmentsOpt != nil {
		hsl = loopers.MakeHighlightSegments(&d.strl, &d.bgl, &d.dl, d.HighlightSegmentsOpt)
	}
	var ccl loopers.ColorizeColor
	if d.ColorizeOpt != nil {
		ccl = loopers.MakeColorizeColor(&d.dl, &d.cl)
	}
	d.eel = loopers.MakeEarlyExit(&d.strl, fixed.I(unpaddedBounds.Size().Y))

	// iteration order
	scl.SetOuterLooper(d.pdl.OuterLooper())
	var outer loopers.Looper = &scl
	if d.ColorizeOpt != nil {
		ccl.SetOuterLooper(outer)
		outer = &ccl
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
	if d.FlashSelectionOpt != nil {
		fsl.SetOuterLooper(outer)
		outer = &fsl
	}
	if d.WrapLineOpt != nil {
		d.wlinecl.SetOuterLooper(outer)
		outer = &d.wlinecl
	}
	// bg phase last looper
	d.bgl.SetOuterLooper(outer)
	// rune phase
	if d.CursorIndex != nil {
		cursorl.SetOuterLooper(outer)
		outer = &cursorl
	}
	d.dl.SetOuterLooper(outer)
}

func (d *HSDrawer) LineHeight() int {
	// strl not initialized
	if d.strl == (loopers.String{}) {
		return 0
	}

	return d.strl.LineHeight().Ceil()
}

func (d *HSDrawer) GetPoint(index int) image.Point {
	if !d.pdl.Initialized {
		return image.Point{}
	}

	d.initMeasurers(d.args.maxX)

	d.pdl.RestorePosDataCloseToIndex(index)
	p := d.pdl.GetPoint(index)                // minimum of pen bounds
	p2 := image.Point{p.X.Ceil(), p.Y.Ceil()} // equal or inside the pen bounds
	return p2
}
func (d *HSDrawer) GetIndex(p *image.Point) int {
	if !d.pdl.Initialized {
		return 0
	}

	d.initMeasurers(d.args.maxX)

	p2 := fixed.P(p.X, p.Y)
	d.pdl.RestorePosDataCloseToPoint(&p2)
	return d.pdl.GetIndex(&p2)
}
