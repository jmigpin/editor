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
	Face font.Face
	Str  string

	Colors      *Colors
	CursorIndex int // <0 to disable
	HWordIndex  int // <0 to disable
	Selection   *loopers.SelectionIndexes
	OffsetY     int
	Pad         image.Point // left/top pad

	height int

	strl    loopers.StringLooper
	wlinel  loopers.WrapLineLooper
	pdl     loopers.PosDataLooper
	wlinecl loopers.WrapLineColorLooper
	dl      loopers.DrawLooper
	bgl     loopers.BgLooper
	eel     loopers.EarlyExitLooper

	maxX int
}

func NewHSDrawer(face font.Face) *HSDrawer {
	d := &HSDrawer{Face: face}

	// Needs strl initialized with face to answer to d.LineHeight
	d.strl.Init(d.Face, d.Str)

	// small pad added to allow the cursor to be fully drawn on first position
	d.Pad = image.Point{1, 0}

	return d
}

func (d *HSDrawer) Measure(max image.Point) image.Point {

	//if d.hintStr == d.Str {
	//	return d.update(hint)
	//}

	// fixed.Int26_6 integer part ranges from -33554432 to 33554431
	//fixedMaxY := fixed.I(33554431).Ceil()

	d.maxX = max.X
	unpaddedMaxX := d.maxX - d.Pad.X

	// loopers
	d.pdl.Init()
	d.initMeasureLoopers(unpaddedMaxX)
	ml := loopers.NewMeasureLooper(&d.strl)

	// iterator order
	ml.SetOuterLooper(&d.pdl)

	// run measure
	ml.Loop(func() bool { return true })
	m := image.Point{ml.M.X.Ceil(), ml.M.Y.Ceil()}

	// keep string height
	d.height = m.Y
	if d.Str == "" {
		d.height = 0
	}

	// add pad and truncate measure
	m = m.Add(d.Pad)
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
	d.initDrawLoopers(img, bounds)

	// restore position to a close data point (performance)
	p := &fixed.Point26_6{0, fixed.I(d.OffsetY)}
	d.pdl.RestorePosDataCloseToPoint(p)
	d.strl.Pen.Y -= fixed.I(d.OffsetY)

	// draw bg
	d.eel.SetOuterLooper(&d.bgl)
	d.eel.Loop(func() bool { return true })

	// prepare for draw runes
	d.initDrawLoopers(img, bounds)

	// restore position to a close data point (performance)
	d.pdl.RestorePosDataCloseToPoint(p)
	d.strl.Pen.Y -= fixed.I(d.OffsetY)

	// draw runes
	d.eel.SetOuterLooper(&d.dl)
	d.eel.Loop(func() bool { return true })
}

func (d *HSDrawer) initMeasureLoopers(maxX int) {
	fmaxX := fixed.I(maxX)

	d.strl.Init(d.Face, d.Str)
	linel := loopers.NewLineLooper(&d.strl)
	d.wlinel.Init(&d.strl, linel, fmaxX)
	d.pdl.Setup(&d.strl, []loopers.PosDataKeeper{&d.strl, &d.wlinel})

	// iterator order
	start := &loopers.EmbedLooper{}
	d.strl.SetOuterLooper(start)
	linel.SetOuterLooper(&d.strl)
	d.wlinel.SetOuterLooper(linel)
	d.pdl.SetOuterLooper(&d.wlinel)
}
func (d *HSDrawer) initDrawLoopers(img draw.Image, bounds *image.Rectangle) {
	// use bounds without the pad for drawing runes, the cursor draws on full bounds
	u := *bounds
	u.Min = u.Min.Add(d.Pad)
	unpaddedBounds := &u

	// loopers
	d.initMeasureLoopers(unpaddedBounds.Size().X)
	d.dl.Init(&d.strl, img, unpaddedBounds)
	d.bgl.Init(&d.strl, &d.dl)
	sl := loopers.NewSelectionLooper(&d.strl, &d.bgl, &d.dl)
	cursorl := loopers.NewCursorLooper(&d.strl, &d.dl, bounds)
	scl := loopers.NewSetColorsLooper(&d.dl, &d.bgl)
	hwl := loopers.NewHWordLooper(&d.strl, &d.bgl, &d.dl)
	d.wlinecl.Init(&d.wlinel, &d.dl, &d.bgl)
	d.eel.Init(&d.strl, fixed.I(unpaddedBounds.Size().Y))

	// options
	d.dl.Fg = d.Colors.Normal.Fg // default fg on which the cursor looks at. If there are nothing to draw, then this never gets set by the scl, and the cursor needs this to be non-nil
	scl.Fg = d.Colors.Normal.Fg
	scl.Bg = nil // d.Colors.Normal.Bg // default bg filled externallly
	sl.Selection = d.Selection
	sl.Fg = d.Colors.Selection.Fg
	sl.Bg = d.Colors.Selection.Bg
	hwl.WordIndex = d.HWordIndex
	hwl.Fg = d.Colors.Highlight.Fg
	hwl.Bg = d.Colors.Highlight.Bg
	cursorl.CursorIndex = d.CursorIndex
	d.wlinecl.Fg = d.Colors.WrapLine.Fg
	d.wlinecl.Bg = d.Colors.WrapLine.Bg

	// iteration order
	scl.SetOuterLooper(&d.wlinel)
	sl.SetOuterLooper(scl)
	hwl.SetOuterLooper(sl)
	d.wlinecl.SetOuterLooper(hwl)
	d.bgl.SetOuterLooper(&d.wlinecl)   // bg phase
	cursorl.SetOuterLooper(&d.wlinecl) // rune phase
	d.dl.SetOuterLooper(cursorl)
}

func (d *HSDrawer) Height() int {
	return d.height
}
func (d *HSDrawer) LineHeight() int {
	return d.strl.LineHeight().Ceil()
}

func (d *HSDrawer) GetPoint(index int) image.Point {
	unpaddedMaxX := d.maxX - d.Pad.X
	d.initMeasureLoopers(unpaddedMaxX)

	d.pdl.RestorePosDataCloseToIndex(index)
	p := d.pdl.GetPoint(index, &d.wlinel)     // minimum of pen bounds
	p2 := image.Point{p.X.Ceil(), p.Y.Ceil()} // equal or inside the pen bounds
	return p2.Add(d.Pad)
}
func (d *HSDrawer) GetIndex(p *image.Point) int {
	unpaddedMaxX := d.maxX - d.Pad.X
	d.initMeasureLoopers(unpaddedMaxX)

	p2 := p.Sub(d.Pad)
	p3 := fixed.P(p2.X, p2.Y)
	d.pdl.RestorePosDataCloseToPoint(&p3)
	return d.pdl.GetIndex(&p3, &d.wlinel)
}
