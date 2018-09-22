package drawer3

//import (
//	"image"
//	"image/color"
//	"image/draw"
//	"strings"

//	"github.com/jmigpin/editor/util/iout"
//	"github.com/jmigpin/editor/util/mathutil"
//	"golang.org/x/image/font"
//)

//type SDrawer struct {
//	Args   args // args that affect cache invalidation. Set externally only.
//	Offset image.Point
//	Fg     color.Color // default foreground color

//	// extensions that don't keep state
//	CursorOpt   *CursorOpt
//	SegmentsOpt *SegmentsOpt

//	cargs args // current args

//	measurer *SMeasurer

//	// extensions
//	exts     []Ext
//	ee       EarlyExit
//	cc       CurColors
//	segs     Segments
//	curs     Cursor
//	bgf      BgFill
//	dru      DrawRune
//	wlinec   WrapLineColor
//	csyntaxc ColorizeSyntaxColor
//}

//func NewSDrawer() *SDrawer {
//	d := &SDrawer{}

//	d.segs.cc = &d.cc
//	d.bgf.cc = &d.cc
//	d.dru.cc = &d.cc
//	d.wlinec.cc = &d.cc
//	d.csyntaxc.cc = &d.cc
//	d.exts = []Ext{&d.ee, &d.cc, &d.csyntaxc, &d.segs, &d.wlinec, &d.bgf, &d.dru, &d.curs}

//	return d
//}

////----------

//func (d *SDrawer) FullMeasure() mathutil.Point64 {
//	if d.measurer == nil {
//		return mathutil.Point64{}
//	}
//	return d.measurer.m.measure
//}

//func (d *SDrawer) LineHeight() int {
//	if d.cargs.Face == nil {
//		return 0
//	}
//	m := d.cargs.Face.Metrics()
//	return LineHeightInt(&m)
//}

////----------

//func (d *SDrawer) NeedMeasure() bool {
//	return d.Args != d.cargs || d.measurer == nil
//}

//func (d *SDrawer) updateArgs() {
//	if d.Args.Str != d.cargs.Str {
//		d.Args.R = strings.NewReader(d.Args.Str)
//	}

//	d.cargs = d.Args
//}

//func (d *SDrawer) ready() bool {
//	return d.cargs == d.Args && d.cargs.Face != nil && d.cargs.R != nil
//}

////----------

//func (d *SDrawer) Measure(max image.Point) image.Point {
//	d.Args.SetMaxX(max.X)
//	if !d.NeedMeasure() {
//		// TODO
//		//return imageutil.MinPoint(d.measurer.m.measure, max)
//		return image.Point{}
//	}

//	d.updateArgs()

//	if !d.ready() {
//		return image.Point{}
//	}

//	d.measurer = NewMeasurer()

//	d.measurer.initExts(d)
//	d.measurer.m.SetOn(true)
//	d.measurer.pd.SetOn(true)
//	// disable these exts at the end, they are just iterated here
//	defer d.measurer.m.SetOn(false)
//	defer d.measurer.pd.SetOn(false)

//	//RunExts(d, &d.measurer.rr, d.measurer.exts)

//	// TODO
//	//return imageutil.MinPoint(d.measurer.m.measure, max)
//	return image.Point{}
//}

////----------

//func (d *SDrawer) Draw(img draw.Image, bounds *image.Rectangle, fg color.Color) {
//	d.Args.SetMaxX(bounds.Size().X)
//	if d.NeedMeasure() {
//		// TODO: warning msg
//		//_ = d.Measure(bounds.Size())
//	}

//	if !d.ready() {
//		return
//	}

//	d.measurer.initExts(d)

//	// restore position to a close data point (performance)
//	//o := fixed.P(d.Offset.X, d.Offset.Y)
//	//d.measurer.pd.RestoreCloseToPoint(&o)

//	//d.initExts(img, bounds)
//	//exts2 := append(d.measurer.exts, d.exts...)

//	//RunExts(d, &d.measurer.rr, exts2)
//}

////----------

//func (d *SDrawer) initExts(img draw.Image, bounds *image.Rectangle) {
//	// early exit
//	//w := bounds.Max.Add(d.Offset)
//	//d.ee.maxY = fixed.I(w.Y)

//	d.bgf.img = img
//	d.bgf.bounds = bounds

//	d.dru.img = img
//	d.dru.bounds = bounds

//	if d.SegmentsOpt == nil {
//		d.segs.SetOn(false)
//	} else {
//		d.segs.SetOn(true)
//		d.segs.opt = d.SegmentsOpt
//	}

//	//if d.CursorOpt == nil {
//	//	d.curs.SetOn(false)
//	//} else {
//	//	d.curs.SetOn(true)
//	//	d.curs.opt = d.CursorOpt
//	//	d.curs.img = img
//	//	d.curs.bounds = bounds
//	//}

//	if !d.measurer.wline.On() {
//		d.wlinec.SetOn(false)
//	} else {
//		d.wlinec.SetOn(true)
//		d.wlinec.wline = &d.measurer.wline
//	}

//	if !d.measurer.csyntax.On() {
//		d.csyntaxc.SetOn(false)
//	} else {
//		d.csyntaxc.SetOn(true)
//		d.csyntaxc.csyntax = &d.measurer.csyntax
//	}
//}

////----------

//func (d *SDrawer) GetPoint(index int) image.Point {
//	if d.measurer == nil {
//		return image.Point{}
//	}
//	return d.measurer.GetPoint(index, d)
//}

//func (d *SDrawer) GetIndex(p image.Point) int {
//	if d.measurer == nil {
//		return 0
//	}
//	return d.measurer.GetIndex(p, d)
//}

////----------

//func (d *SDrawer) GetOffsetPoint(index int) image.Point {
//	p := d.GetPoint(index)
//	return p.Sub(d.Offset)
//}

//func (d *SDrawer) GetOffsetIndex(p image.Point) int {
//	p2 := p.Add(d.Offset)
//	return d.GetIndex(p2)
//}

////----------

//// arguments that must match to allow the cached calculations to persist
//type args struct {
//	Str string // TODO

//	R                iout.ReadSeekRuner
//	Face             font.Face
//	FirstLineOffsetX int

//	// extensions that keep state and affect cache need to be recalculated
//	WrapLineOpt       *WrapLineOpt
//	ColorizeSyntaxOpt *ColorizeSyntaxOpt

//	wrapLineMaxX int
//}

//func (a *args) SetMaxX(maxX int) {
//	if a.WrapLineOpt != nil {
//		a.wrapLineMaxX = maxX
//	}
//}

////----------

//type SMeasurer struct {
//	exts []Ext

//	// extentions
//	rr      RuneReader
//	line    Line
//	wline   WrapLine
//	csyntax ColorizeSyntax
//	pd      PosData
//	m       Measure

//	// opt
//	wlineOpt   WrapLineOpt
//	csyntaxOpt ColorizeSyntaxOpt
//}

//func NewMeasurer() *SMeasurer {
//	m := &SMeasurer{}

//	m.wline.line = &m.line

//	m.pd.jump = 300
//	m.pd.keepers = []PosDataKeeper{&m.rr, &m.line, &m.wline}

//	m.exts = []Ext{&m.rr, &m.line, &m.wline, &m.csyntax, &m.pd, &m.m}

//	m.pd.SetOn(false)
//	m.m.SetOn(false)

//	return m
//}

////----------

//func (m *SMeasurer) initExts(d *SDrawer) {
//	m.rr = MakeRuneReader(d.cargs.Face, d.cargs.R)
//	//m.rr.Pen.X = fixed.I(d.cargs.FirstLineOffsetX)
//	//m.rr.reader.Seek(0, io.SeekStart) // TODO: handle error

//	m.wline.data = WrapLineData{}

//	if d.cargs.WrapLineOpt == nil {
//		m.wline.SetOn(false)
//	} else {
//		m.wline.SetOn(true)
//		m.wline.opt = d.cargs.WrapLineOpt
//		m.wline.maxX = mathutil.IntfFromInt(d.cargs.wrapLineMaxX)
//	}

//	if d.cargs.ColorizeSyntaxOpt == nil {
//		m.csyntax.SetOn(false)
//	} else {
//		m.csyntax.SetOn(true)
//		m.csyntax.opt = d.cargs.ColorizeSyntaxOpt
//		m.csyntax.data = ColorizeSyntaxData{}
//	}
//}

////----------

//func (m *SMeasurer) GetPoint(index int, d *SDrawer) image.Point {
//	m.initExts(d)

//	//m.pd.RestoreCloseToIndex(index) // restores keepers exts

//	//gp := GetPoint{index: index}
//	//exts2 := append(m.pd.keepersExts(), &gp)

//	//RunExts(d, &m.rr, exts2)

//	// pen lines were added with lineheight ceil, use floor
//	//p := gp.pen
//	//return image.Point{p.X.Floor(), p.Y.Floor()}
//	return image.Point{}
//}

//func (m *SMeasurer) GetIndex(p image.Point, d *SDrawer) int {
//	//m.initExts(d)

//	//p2 := fixed.P(p.X, p.Y)
//	//m.pd.RestoreCloseToPoint(&p2) // restores keepers exts

//	//gi := GetIndex{point: p2}
//	////exts2 := append(m.pd.keepersExts(), &gi)

//	////RunExts(d, &m.rr, exts2)

//	//return gi.index
//	return 0
//}
