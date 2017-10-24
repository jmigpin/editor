package hsdrawer

import (
	"image"
	"image/draw"

	"github.com/jmigpin/editor/drawutil2/loopers"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Highlight and Selection drawer.
type HSDrawer2 struct {
	Face font.Face
	Str  string

	Colors      *Colors
	CursorIndex int // <0 to disable
	HWordIndex  int // <0 to disable
	Selection   *loopers.SelectionIndexes
	OffsetY     fixed.Int26_6
	Pad         image.Point // left/top pad

	height fixed.Int26_6

	pdl    *loopers.PosDataLooper
	pdk    *HSPosDataKeeper2
	wlinel *loopers.WrapLine2Looper
}

func NewHSDrawer2(face font.Face) *HSDrawer2 {
	d := &HSDrawer2{Face: face}

	// compute with minimal data
	// allows getpoint to work without a calcrunedata be called
	//d.Measure(&image.Point{})

	// small pad added to allow the cursor to be fully drawn on first position
	d.Pad = image.Point{1, 0}

	return d
}

func (d *HSDrawer2) Measure(max0 *image.Point) *fixed.Point26_6 {
	max := *max0
	max = max.Sub(d.Pad)

	max2 := fixed.P(max.X, max.Y)

	strl := loopers.NewStringLooper(d.Face, d.Str)
	linel := loopers.NewLineLooper(strl, max2.Y)
	wlinel := loopers.NewWrapLine2Looper(strl, linel, max2.X)
	d.pdk = NewHSPosDataKeeper2(wlinel)
	d.pdl = loopers.NewPosDataLooper(strl, d.pdk)
	ml := loopers.NewMeasureLooper(strl, &max2)

	d.wlinel = wlinel

	// iterator order
	linel.SetOuterLooper(strl)
	wlinel.SetOuterLooper(linel)
	d.pdl.SetOuterLooper(wlinel)
	ml.SetOuterLooper(d.pdl)

	ml.Loop(func() bool { return true })

	d.height = ml.M.Y
	if d.Str == "" {
		d.height = 0
	}

	return ml.M
}
func (d *HSDrawer2) Draw(img draw.Image, bounds0 *image.Rectangle) {
	t := *bounds0
	bounds := &t
	bounds.Min = bounds.Min.Add(d.Pad)

	strl := d.pdl.Strl
	wlinel := d.wlinel
	dl := loopers.NewDrawLooper(strl, img, bounds)
	bgl := loopers.NewBgLooper(strl, dl)
	sl := loopers.NewSelectionLooper(strl, bgl, dl)
	cursorl := loopers.NewCursorLooper(strl, dl, bounds0)
	scl := loopers.NewSetColorsLooper(dl, bgl)
	hwl := loopers.NewHWordLooper(strl, bgl, dl)
	wlinecl := loopers.NewWrapLine2ColorLooper(wlinel, dl, bgl)
	eel := loopers.NewEarlyExitLooper(strl, bounds)

	// options
	scl.Fg = d.Colors.Normal.Fg
	scl.Bg = nil // d.Colors.Normal.Bg // default bg filled externallly
	sl.Selection = d.Selection
	sl.Fg = d.Colors.Selection.Fg
	sl.Bg = d.Colors.Selection.Bg
	hwl.WordIndex = d.HWordIndex
	hwl.Fg = d.Colors.Highlight.Fg
	hwl.Bg = d.Colors.Highlight.Bg
	cursorl.CursorIndex = d.CursorIndex

	// draw bg first to correctly paint below all runes drawn later

	// bg iteration order
	scl.SetOuterLooper(wlinel)
	sl.SetOuterLooper(scl)
	hwl.SetOuterLooper(sl)
	wlinecl.SetOuterLooper(hwl)
	bgl.SetOuterLooper(wlinecl)
	eel.SetOuterLooper(bgl)

	// restore position to a close data point (performance)
	p := &fixed.Point26_6{0, d.OffsetY}
	d.pdl.RestorePosDataCloseToPoint(p)
	d.pdl.Strl.Pen.Y -= d.OffsetY

	// draw bg
	eel.Loop(func() bool { return true })

	// options

	// iterator order
	scl.SetOuterLooper(wlinel)
	sl.SetOuterLooper(scl)
	hwl.SetOuterLooper(sl)
	wlinecl.SetOuterLooper(hwl)
	cursorl.SetOuterLooper(wlinecl)
	dl.SetOuterLooper(cursorl)
	eel.SetOuterLooper(dl)

	// restore position to a close data point (performance)
	d.pdl.RestorePosDataCloseToPoint(p)
	d.pdl.Strl.Pen.Y -= d.OffsetY

	// draw runes
	eel.Loop(func() bool { return true })
}

func (d *HSDrawer2) Height() fixed.Int26_6 {
	return d.height
}
func (d *HSDrawer2) LineHeight() fixed.Int26_6 {
	// TODO: remove this check
	if d.pdl == nil {
		return 0
	}

	return d.pdl.Strl.LineHeight()
}

func (d *HSDrawer2) GetPoint(index int) *fixed.Point26_6 {
	// TODO: remove this check
	if d.pdl == nil {
		return &fixed.Point26_6{}
	}

	d.pdl.RestorePosDataCloseToIndex(index)
	p := d.pdl.GetPoint(index, d.wlinel)
	p2 := p.Add(fixed.P(d.Pad.X, d.Pad.Y))
	return &p2
}
func (d *HSDrawer2) GetIndex(p *fixed.Point26_6) int {
	// TODO: remove this check
	if d.pdl == nil {
		return 0
	}

	p2 := p.Sub(fixed.P(d.Pad.X, d.Pad.Y))
	d.pdl.RestorePosDataCloseToPoint(&p2)
	return d.pdl.GetIndex(&p2, d.wlinel)
}

type HSPosDataKeeper2 struct {
	wlinel *loopers.WrapLine2Looper
}

func NewHSPosDataKeeper2(wlinel *loopers.WrapLine2Looper) *HSPosDataKeeper2 {
	return &HSPosDataKeeper2{wlinel: wlinel}
}
func (pdk *HSPosDataKeeper2) KeepPosData() interface{} {
	return pdk.wlinel.WrapData
}
func (pdk *HSPosDataKeeper2) RestorePosData(data interface{}) {
	pdk.wlinel.WrapData = data.(loopers.WrapLine2Data)
}
