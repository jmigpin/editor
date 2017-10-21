package hsdrawer

import (
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
	OffsetY     fixed.Int26_6
	Pad         image.Point // left/top pad

	height fixed.Int26_6

	pdl    *loopers.PosDataLooper
	pdk    *HSPosDataKeeper
	wlinel *loopers.WrapLineLooper
}

func NewHSDrawer(face font.Face) *HSDrawer {
	d := &HSDrawer{Face: face}

	// compute with minimal data
	// allows getpoint to work without a calcrunedata be called
	//d.Measure(&image.Point{})

	// small pad added to allow the cursor to be fully drawn on first position
	d.Pad = image.Point{1, 0}

	return d
}

func (d *HSDrawer) Measure(max0 *image.Point) *fixed.Point26_6 {
	max := *max0
	max = max.Sub(d.Pad)

	max2 := fixed.P(max.X, max.Y)

	strl := loopers.NewStringLooper(d.Face, d.Str)
	linel := loopers.NewLineLooper(strl, max2.Y)
	wlinel := loopers.NewWrapLineLooper(strl, linel, max2.X)
	d.pdk = NewHSPosDataKeeper(wlinel)
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
func (d *HSDrawer) Draw(img draw.Image, bounds0 *image.Rectangle) {
	t := *bounds0
	bounds := &t
	bounds.Min = bounds.Min.Add(d.Pad)

	strl := d.pdl.Strl
	wlinel := d.wlinel
	dl := loopers.NewDrawLooper(strl, img, bounds)
	bgl := loopers.NewBgLooper(strl, dl)
	sl := loopers.NewSelectionLooper(strl, bgl, dl)
	cursorl := loopers.NewCursorLooper(strl, dl, bounds0)
	hwl := loopers.NewHWordLooper(strl, bgl, dl, sl)
	scl := loopers.NewSetColorsLooper(dl, bgl)
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

	// draw background first to correctly paint letters above the background

	// bg iteration order
	scl.SetOuterLooper(wlinel)
	sl.SetOuterLooper(scl)
	hwl.SetOuterLooper(sl)
	bgl.SetOuterLooper(hwl)
	eel.SetOuterLooper(bgl)

	// restore position to a close data point (performance)
	p := &fixed.Point26_6{0, d.OffsetY}
	d.pdl.RestorePosDataCloseToPoint(p)
	d.pdl.Strl.Pen.Y -= d.OffsetY

	// draw bg
	eel.Loop(func() bool { return true })

	// iterator order
	cursorl.SetOuterLooper(wlinel)
	dl.SetOuterLooper(cursorl)
	eel.SetOuterLooper(dl)

	// restore position to a close data point (performance)
	d.pdl.RestorePosDataCloseToPoint(p)
	d.pdl.Strl.Pen.Y -= d.OffsetY

	// draw runes
	eel.Loop(func() bool { return true })
}

func (d *HSDrawer) Height() fixed.Int26_6 {
	return d.height
}
func (d *HSDrawer) LineHeight() fixed.Int26_6 {
	// TODO: remove this check
	if d.pdl == nil {
		return 0
	}

	return d.pdl.Strl.LineHeight()
}

func (d *HSDrawer) GetPoint(index int) *fixed.Point26_6 {
	// TODO: remove this check
	if d.pdl == nil {
		return &fixed.Point26_6{}
	}

	d.pdl.RestorePosDataCloseToIndex(index)
	p := d.pdl.GetPoint(index, d.wlinel)
	p2 := p.Add(fixed.P(d.Pad.X, d.Pad.Y))
	return &p2
}
func (d *HSDrawer) GetIndex(p *fixed.Point26_6) int {
	// TODO: remove this check
	if d.pdl == nil {
		return 0
	}

	p2 := p.Sub(fixed.P(d.Pad.X, d.Pad.Y))
	d.pdl.RestorePosDataCloseToPoint(&p2)
	return d.pdl.GetIndex(&p2, d.wlinel)
}

type HSPosDataKeeper struct {
	wlinel *loopers.WrapLineLooper
}

func NewHSPosDataKeeper(wlinel *loopers.WrapLineLooper) *HSPosDataKeeper {
	return &HSPosDataKeeper{wlinel: wlinel}
}
func (pdk *HSPosDataKeeper) KeepPosData() interface{} {
	return &HSPosData{
		wrapIndent: pdk.wlinel.WrapIndent,
	}
}
func (pdk *HSPosDataKeeper) RestorePosData(data interface{}) {
	u := data.(*HSPosData)
	pdk.wlinel.WrapIndent = u.wrapIndent
}

type HSPosData struct {
	wrapIndent loopers.WrapIndent
}
