package hsdrawer

import (
	"image"
	"image/draw"

	"github.com/jmigpin/editor/drawutil2/loopers"
	"github.com/jmigpin/editor/drawutil2/posindex"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Highlight and Selection drawer.
type HSDrawer struct {
	Face        font.Face
	Str         string
	Colors      *Colors
	CursorIndex int // <0 to disable
	HWordIndex  int // <0 to disable
	Selection   *loopers.SelectionIndexes
	OffsetY     fixed.Int26_6

	pl  HSPosLooper
	pdi *posindex.PosDataIndex
}

func (d *HSDrawer) Measure(max *image.Point) *fixed.Point26_6 {
	max2 := fixed.P(max.X, max.Y)

	strl := loopers.NewStringLooper(d.Face, d.Str)
	linel := loopers.NewLineLooper(strl, max2.Y)
	wlinel := loopers.NewWrapLineLooper(strl, linel, max2.X)
	ml := loopers.NewMeasureLooper(strl, &max2)
	pl := NewHSPosLooper(strl, wlinel)

	pl := &HSPosLooper{strl: strl, wlinel: wlinel}
	d.pl = pl
	d.pdi = posindex.NewPosDataIndex(pl)

	// options
	strl.Pen.Y += d.OffsetY

	// iterator order
	linel.Looper = strl
	wlinel.Looper = linel
	ml.Looper = wlinel
	pl.Looper = ml

	pl.Loop(func() bool { return true })

	return ml.M
}
func (d *HSDrawer) Draw(img draw.Image, bounds *image.Rectangle) {
	min := fixed.P(bounds.Min.X, bounds.Min.Y)
	max := fixed.P(bounds.Max.X, bounds.Max.Y)
	max2 := max.Sub(min)

	//strl := loopers.NewStringLooper(d.Face, d.Str)
	//linel := loopers.NewLineLooper(strl, max2.Y)
	//wlinel := loopers.NewWrapLineLooper(strl, linel, max2.X)
	strl := d.pl.strl
	wlinel := d.pl.wlinel

	dl := loopers.NewDrawLooper(strl, img, bounds)
	bgl := loopers.NewBgLooper(strl, dl)
	sl := loopers.NewSelectionLooper(strl, bgl, dl)
	cursorl := loopers.NewCursorLooper(strl, dl)
	hwl := loopers.NewHWordLooper(strl, bgl, dl, sl)
	scl := loopers.NewSetColorsLooper(dl, bgl)

	// options
	strl.Pen.Y += d.OffsetY
	scl.Fg = d.Colors.Normal.Fg
	scl.Bg = d.Colors.Normal.Bg
	sl.Selection = d.Selection
	sl.Fg = d.Colors.Selection.Fg
	sl.Bg = d.Colors.Selection.Bg
	hwl.WordIndex = d.HWordIndex
	hwl.Fg = d.Colors.Highlight.Fg
	hwl.Bg = d.Colors.Highlight.Bg
	cursorl.CursorIndex = d.CursorIndex

	// iterator order
	//linel.Looper = strl
	//wlinel.Looper = linel
	scl.Looper = wlinel
	sl.Looper = scl
	hwl.Looper = sl
	bgl.Looper = hwl
	cursorl.Looper = bgl
	dl.Looper = cursorl

	dl.Loop(func() bool { return true })
}

func (d *HSDrawer) Height() fixed.Int26_6 {
}
func (d *HSDrawer) GetIndex(p *fixed.Point26_6) int {
	d.pdi.RestorePosDataCloseToPoint(p)
	posindex.NewPointIndexPos(d.pl)

	pointindexpos
}
func (d *HSDrawer) GetPoint(index int) *fixed.Point26_6 {
	d.pdi.RestorePosDataCloseToIndex(index)
}

type HSPosLooper struct {
	Looper loopers.Looper
	strl   *loopers.StringLooper
	wlinel *loopers.WrapLineLooper
	d.pdi = posindex.NewPosDataIndex(pl)
}

func NewHSPosLooper(strl *loopers.StringLooper, wlinel *loopers.WrapLineLooper) *HSPosLooper {
	pl:=&HSPosLooper{strl: strl, wlinel: wlinel}
	pl.pdi = posindex.NewPosDataIndex(pl)
	return pl
}

func (pl *HSPosLooper) Loop(fn func() bool) {
	pl.Looper.Loop(fn)
}

func (pl *HSPosLooper) KeepPosData() *posindex.PosData {
	data := &HSPosData{
		ri:         pl.strl.Ri,
		pen:        pl.strl.Pen,
		wrapIndent: pl.wlinel.WrapIndent,
	}
	return &posindex.PosData{
		Index:        pl.strl.Ri,
		PenBoundsMin: pl.strl.PenBounds().Min,
		Data:         data,
	}
}
func (pl *HSPosLooper) RestorePosData(pd *posindex.PosData) {
	data := pd.Data.(*HSPosData)
	pl.strl.Ri = data.ri
	pl.strl.Pen = data.pen
	pl.wlinel.WrapIndent = data.wrapIndent
}

type HSPosData struct {
	ri         int
	pen        fixed.Point26_6
	wrapIndent loopers.WrapIndent
}
