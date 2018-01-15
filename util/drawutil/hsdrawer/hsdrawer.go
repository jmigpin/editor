package hsdrawer

import (
	"image"
	"image/color"
	"image/draw"
	"log"

	"github.com/jmigpin/editor/util/drawutil/loopers"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Highlight and Selection drawer. Has no constructor but needs Face and Str set to measure/draw.
type HSDrawer struct {
	Face             font.Face
	Str              string
	Offset           image.Point
	FirstLineOffsetX int
	Measurement      image.Point
	Fg               color.Color

	// extensions
	CursorIndex          *int
	WrapLineColorOpt     *loopers.WrapLineColorOpt
	HighlightWordOpt     *loopers.HighlightWordOpt
	SelectionOpt         *loopers.SelectionOpt
	FlashSelectionOpt    *loopers.FlashSelectionOpt
	HighlightSegmentsOpt *loopers.HighlightSegmentsOpt
	CommentsOpt          *loopers.CommentsOpt

	strl    loopers.String
	wlinel  loopers.WrapLine
	pdl     loopers.PosData
	wlinecl loopers.WrapLineColor
	dl      loopers.Draw
	bgl     loopers.Bg
	cl      loopers.Comments
	eel     loopers.EarlyExit

	maxX int

	// TODO: left/top pad to help cursor looper to be fully draw
	// small pad added to allow the cursor to be fully drawn on first position
	// need to update getindex/getpoint as well
	//pad image.Point
}

func (d *HSDrawer) Measure(max image.Point) image.Point {
	if d.Face == nil {
		return image.Point{}
	}

	// TODO: need to check early exit looper
	// fixed.Int26_6 integer part ranges from -33554432 to 33554431
	//fixedMaxY := fixed.I(33554431).Ceil()

	d.maxX = max.X

	// loopers
	d.pdl.Data = nil // reset data
	d.initMeasurers(d.maxX)
	ml := loopers.NewMeasure(&d.strl)

	// iterator order
	ml.SetOuterLooper(&d.pdl)

	// run measure
	ml.Loop(func() bool { return true })
	m := image.Point{ml.M.X.Ceil(), ml.M.Y.Ceil()}
	d.Measurement = m

	// truncate measure for return
	if m.X > max.X {
		m.X = max.X
	}
	if m.Y > max.Y {
		m.Y = max.Y
	}

	return m
}

func (d *HSDrawer) Draw(img draw.Image, bounds *image.Rectangle) {
	if d.Face == nil {
		return
	}

	if bounds.Size().X != d.maxX {
		log.Printf("hsdrawer: drawing for %v but measured with hint %v", bounds.Size().X, d.maxX)
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
	if d.WrapLineColorOpt != nil {
		d.wlinel = loopers.MakeWrapLine(&d.strl, &linel, fmaxX)
		keepers = append(keepers, &d.wlinel)
	}
	if d.CommentsOpt != nil {
		d.cl = loopers.MakeComments(&d.strl, d.CommentsOpt)
		keepers = append(keepers, &d.cl)
	}
	d.pdl = loopers.MakePosData(&d.strl, keepers, 250, d.pdl.Data)

	// iterator order
	start := &loopers.EmbedLooper{}
	d.strl.SetOuterLooper(start)
	linel.SetOuterLooper(&d.strl)
	var outer loopers.Looper = &linel
	if d.WrapLineColorOpt != nil {
		d.wlinel.SetOuterLooper(outer)
		outer = &d.wlinel
	}
	if d.CommentsOpt != nil {
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
	if d.WrapLineColorOpt != nil {
		d.wlinecl = loopers.MakeWrapLineColor(&d.wlinel, &d.dl, &d.bgl, d.WrapLineColorOpt)
	}
	var hsl loopers.HighlightSegments
	if d.HighlightSegmentsOpt != nil {
		hsl = loopers.MakeHighlightSegments(&d.strl, &d.bgl, &d.dl, d.HighlightSegmentsOpt)
	}
	var ccl loopers.CommentsColor
	if d.CommentsOpt != nil {
		ccl = loopers.MakeCommentsColor(&d.dl, &d.cl)
	}
	d.eel = loopers.MakeEarlyExit(&d.strl, fixed.I(unpaddedBounds.Size().Y))

	// iteration order
	scl.SetOuterLooper(d.pdl.OuterLooper())
	var outer loopers.Looper = &scl
	if d.CommentsOpt != nil {
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
	if d.WrapLineColorOpt != nil {
		d.wlinecl.SetOuterLooper(outer)
		outer = &d.wlinecl
	}
	// bg phase
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

	d.initMeasurers(d.maxX)

	d.pdl.RestorePosDataCloseToIndex(index)
	p := d.pdl.GetPoint(index)                // minimum of pen bounds
	p2 := image.Point{p.X.Ceil(), p.Y.Ceil()} // equal or inside the pen bounds
	return p2
}
func (d *HSDrawer) GetIndex(p *image.Point) int {
	if !d.pdl.Initialized {
		return 0
	}

	d.initMeasurers(d.maxX)

	p2 := fixed.P(p.X, p.Y)
	d.pdl.RestorePosDataCloseToPoint(&p2)
	return d.pdl.GetIndex(&p2)
}
