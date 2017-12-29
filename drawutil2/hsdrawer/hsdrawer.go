package hsdrawer

import (
	"fmt"
	"image"
	"image/draw"

	"github.com/jmigpin/editor/drawutil2/loopers"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Highlight and Selection drawer.
type HSDrawer struct {
	Face             font.Face
	Str              string
	Colors           *Colors
	CursorIndex      *int
	HWordIndex       *int
	Selection        *loopers.SelectionIndexes
	FlashSelection   *loopers.FlashSelectionIndexes
	Offset           image.Point
	Pad              image.Point // left/top pad
	FirstLineOffsetX int

	EnableWrapLine bool
	Measurement    image.Point

	maxX int

	strl    loopers.String
	wlinel  loopers.WrapLine
	pdl     loopers.PosData
	wlinecl loopers.WrapLineColor
	dl      loopers.Draw
	bgl     loopers.Bg
	eel     loopers.EarlyExit
}

func NewHSDrawer(face font.Face) *HSDrawer {
	d := &HSDrawer{Face: face}

	// Needs strl initialized with face to answer to d.LineHeight
	d.strl = loopers.MakeString(d.Face, d.Str)

	// small pad added to allow the cursor to be fully drawn on first position
	d.Pad = image.Point{0, 0}

	//d.EnableWrapLine = true

	return d
}

func (d *HSDrawer) Measure(max image.Point) image.Point {

	// TODO: remove pad from offset?

	// TODO: update only parts of the cache
	//if d.hintStr == d.Str {
	//	return d.update(hint)
	//}

	// fixed.Int26_6 integer part ranges from -33554432 to 33554431
	//fixedMaxY := fixed.I(33554431).Ceil()

	d.maxX = max.X
	unpaddedMaxX := d.maxX - d.Pad.X

	// loopers
	d.pdl = loopers.MakePosData()
	d.initMeasures(unpaddedMaxX)
	ml := loopers.NewMeasure(&d.strl)

	// iterator order
	ml.SetOuterLooper(&d.pdl)

	// run measure
	ml.Loop(func() bool { return true })
	m := image.Point{ml.M.X.Ceil(), ml.M.Y.Ceil()}

	// add pad
	m = m.Add(d.Pad)
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
	if bounds.Size().X != d.maxX {
		panic(fmt.Sprintf("drawing for %v but measured with hint %v", bounds.Size().X, d.maxX))
	}

	// draw bg first to correctly paint below all runes drawn later
	d.initDraws(img, bounds)

	// restore position to a close data point (performance)
	p := fixed.P(d.Offset.X, d.Offset.Y)
	//p := &fixed.Point26_6{0, fixed.I(d.Offset.Y)}
	d.pdl.RestorePosDataCloseToPoint(&p)
	//d.strl.Pen.Y -= fixed.I(d.Offset.Y)
	d.strl.Pen = d.strl.Pen.Sub(p)

	// draw bg
	d.eel.SetOuterLooper(&d.bgl)
	d.eel.Loop(func() bool { return true })

	// prepare for draw runes
	d.initDraws(img, bounds)

	// restore position to a close data point (performance)
	d.pdl.RestorePosDataCloseToPoint(&p)
	//d.strl.Pen.Y -= fixed.I(d.Offset.Y)
	d.strl.Pen = d.strl.Pen.Sub(p)

	// draw runes
	d.eel.SetOuterLooper(&d.dl)
	d.eel.Loop(func() bool { return true })
}

func (d *HSDrawer) initMeasures(maxX int) {
	fmaxX := fixed.I(maxX)

	d.strl = loopers.MakeString(d.Face, d.Str)
	d.strl.Pen.X = fixed.I(d.FirstLineOffsetX)
	linel := loopers.MakeLine(&d.strl, fixed.I(d.Offset.X))
	keepers := []loopers.PosDataKeeper{&d.strl}
	if d.EnableWrapLine {
		d.wlinel = loopers.MakeWrapLine(&d.strl, &linel, fmaxX)
		keepers = append(keepers, &d.wlinel)
	}
	d.pdl.Setup(&d.strl, keepers)

	// iterator order
	start := &loopers.EmbedLooper{}
	d.strl.SetOuterLooper(start)
	linel.SetOuterLooper(&d.strl)
	d.pdl.SetOuterLooper(&linel)
	if d.EnableWrapLine {
		d.wlinel.SetOuterLooper(&linel)
		d.pdl.SetOuterLooper(&d.wlinel)
	}
}

func (d *HSDrawer) initDraws(img draw.Image, bounds *image.Rectangle) {
	// use bounds without the pad for drawing runes, the cursor draws on full bounds
	u := *bounds
	u.Min = u.Min.Add(d.Pad)
	unpaddedBounds := &u

	// loopers
	d.initMeasures(unpaddedBounds.Size().X)
	d.dl = loopers.MakeDraw(&d.strl, img, unpaddedBounds)
	d.bgl = loopers.MakeBg(&d.strl, &d.dl)
	sl := loopers.MakeSelection(&d.strl, &d.bgl, &d.dl)
	scl := loopers.NewSetColors(&d.dl, &d.bgl)
	hwl := loopers.MakeHWord(&d.strl, &d.bgl, &d.dl)
	fsl := loopers.NewFlashSelection(&d.strl, &d.bgl, &d.dl)
	cursorl := loopers.NewCursor(&d.strl, &d.dl, bounds)
	if d.EnableWrapLine {
		d.wlinecl = loopers.MakeWrapLineColor(&d.wlinel, &d.dl, &d.bgl)
	}
	d.eel = loopers.MakeEarlyExit(&d.strl, fixed.I(unpaddedBounds.Size().Y))

	// if nil colors are allowed, they should be dealt with here

	// options
	d.bgl.NoColorizeBg = d.Colors.Normal.Bg // filled externally so don't colorize here if it is this color
	scl.Fg = d.Colors.Normal.Fg
	scl.Bg = d.Colors.Normal.Bg
	sl.Selection = d.Selection
	sl.Fg = d.Colors.Selection.Fg
	sl.Bg = d.Colors.Selection.Bg
	hwl.WordIndex = d.HWordIndex
	hwl.Fg = d.Colors.Highlight.Fg
	hwl.Bg = d.Colors.Highlight.Bg
	fsl.Selection = d.FlashSelection
	cursorl.CursorIndex = d.CursorIndex
	if d.EnableWrapLine {
		d.wlinecl.Fg = d.Colors.WrapLine.Fg
		d.wlinecl.Bg = d.Colors.WrapLine.Bg
	}

	// iteration order
	scl.SetOuterLooper(d.pdl.OuterLooper())
	sl.SetOuterLooper(scl)
	hwl.SetOuterLooper(&sl)
	fsl.SetOuterLooper(&hwl)
	if d.EnableWrapLine {
		d.wlinecl.SetOuterLooper(&hwl)
		fsl.SetOuterLooper(&d.wlinecl)
	}
	d.bgl.SetOuterLooper(fsl)   // bg phase
	cursorl.SetOuterLooper(fsl) // rune phase
	d.dl.SetOuterLooper(cursorl)
}

func (d *HSDrawer) LineHeight() int {
	return d.strl.LineHeight().Ceil()
}

func (d *HSDrawer) GetPoint(index int) image.Point {
	unpaddedMaxX := d.maxX - d.Pad.X
	d.initMeasures(unpaddedMaxX)

	d.pdl.RestorePosDataCloseToIndex(index)
	p := d.pdl.GetPoint(index)                // minimum of pen bounds
	p2 := image.Point{p.X.Ceil(), p.Y.Ceil()} // equal or inside the pen bounds
	return p2.Add(d.Pad)
}
func (d *HSDrawer) GetIndex(p *image.Point) int {
	unpaddedMaxX := d.maxX - d.Pad.X
	d.initMeasures(unpaddedMaxX)

	p2 := p.Sub(d.Pad)
	p3 := fixed.P(p2.X, p2.Y)
	d.pdl.RestorePosDataCloseToPoint(&p3)
	return d.pdl.GetIndex(&p3)
}
