package drawer3

import (
	"image"
	"image/color"
	"image/draw"
	"log"

	"github.com/jmigpin/editor/util/mathutil"
)

type PosDrawer struct {
	DrawerCommon

	Cursor         Cursor
	WrapLine       WrapLine
	ColorizeSyntax ColorizeSyntax
	Segments       Segments
	Annotations    Annotations

	mexts []Ext // measure extentions
	dexts []Ext // draw extentions
	pexts []Ext // position extensions (indexof/pointof)

	rr       RuneReader
	pd       PosData
	meas     Measure
	ee       EarlyExit
	line     Line
	wlinec   WrapLineColor
	csyntaxc ColorizeSyntaxColor
	annc     AnnotationsColor
	cc       CurColors
	bgf      BgFill
	dru      DrawRune

	measurement image.Point
}

func NewPosDrawer() *PosDrawer {
	d := &PosDrawer{}

	d.Cursor = Cursor1(&d.cc)
	d.Cursor.SetOn(false)
	d.WrapLine = WrapLine1(&d.line, d)
	d.WrapLine.SetOn(false)
	d.ColorizeSyntax = ColorizeSyntax1(d)
	d.ColorizeSyntax.SetOn(false)
	d.Segments = Segments1(&d.cc, d)
	d.Segments.SetOn(false)
	d.Annotations = Annotations1()
	d.Annotations.SetOn(false)

	// d.rr // no init
	// d.cc // no init
	// d.ee // no init
	// d.line // no init
	d.meas.SetOn(false)
	d.csyntaxc = ColorizeSyntaxColor1(&d.ColorizeSyntax, &d.cc)
	d.wlinec = WrapLineColor1(&d.WrapLine, &d.cc)
	d.annc = AnnotationsColor1(&d.Annotations, &d.cc)
	d.bgf = BgFill1(&d.cc)
	d.dru = DrawRune1(&d.cc)

	keepers := []PosDataKeeper{
		&d.rr,
		&d.WrapLine,
		&d.ColorizeSyntax,
	}
	d.pd = PosData1(10, keepers)
	d.pd.SetOn(false)

	d.pexts = []Ext{
		&d.rr,
		&d.line,
		&d.WrapLine,
		&d.ColorizeSyntax,
		&d.Annotations,
	}
	d.mexts = append(d.pexts, []Ext{
		&d.pd,
		&d.meas,
	}...)
	d.dexts = append(d.mexts, []Ext{
		&d.ee,
		&d.cc,
		&d.csyntaxc,
		&d.Segments,
		&d.wlinec,
		&d.annc,
		&d.bgf,
		&d.dru,
		&d.Cursor,
	}...)

	return d
}

//----------

func (d *PosDrawer) ready() bool {
	return !(d.face == nil || d.reader == nil || d.bounds == image.ZR)
}

//----------

func (d *PosDrawer) Measure() image.Point {
	if !d.ready() {
		return image.Point{}
	}

	if !d.needMeasure {
		return d.measurement
	}
	d.needMeasure = false

	// restores original offset after measuring
	keep := d.WrapLine.On() && d.Offset().Y > 0
	if keep {
		o := d.Offset()
		offsetIndex := d.IndexOf(o)
		p := d.PointOf(offsetIndex)
		offsetYmargin := o.Y - p.Y
		defer func() {
			p := d.PointOf(offsetIndex)
			y := p.Y + offsetYmargin
			if o.Y != y {
				o.Y = y
				d.SetOffset(o)
			}
		}()
	}

	d.pd.SetOn(true)
	defer d.pd.SetOn(false)
	d.meas.SetOn(true)
	defer d.meas.SetOn(false)

	postStart := func() {
		d.WrapLine.data.maxX = mathutil.Intf1(d.Bounds().Dx())
	}

	RunExts(d, &d.rr, d.mexts, postStart)

	d.measurement = d.meas.measure

	return d.measurement
}

//----------

func (d *PosDrawer) Draw(img draw.Image, fg color.Color) {
	if !d.ready() {
		return
	}

	if d.needMeasure {
		log.Printf("warning: draw needmeasure: bounds=%v", d.Bounds())
	}

	d.cc.setup(fg)
	d.Cursor.setup(img)
	d.bgf.setup(img)
	d.dru.setup(img)

	postStart := func() {
		// restore position to a close data point (performance)
		o := mathutil.PIntf2(d.Offset())
		d.pd.RestoreCloseToPoint(o)
	}

	RunExts(d, &d.rr, d.dexts, postStart)
}

//----------

func (d *PosDrawer) PointOf(index int) image.Point {
	if !d.ready() {
		return image.Point{}
	}

	pof := PointOf1(index)
	exts2 := append(d.pexts, &pof)

	postStart := func() {
		d.pd.RestoreCloseToIndex(index) // setups keepers exts
	}

	RunExts(d, &d.rr, exts2, postStart)

	return pof.point
}

func (d *PosDrawer) IndexOf(p image.Point) int {
	if !d.ready() {
		return 0
	}

	p2 := mathutil.PIntf2(p)
	iof := IndexOf1(p2)
	exts2 := append(d.pexts, &iof)

	postStart := func() {
		d.pd.RestoreCloseToPoint(p2) // setups keepers exts
	}

	RunExts(d, &d.rr, exts2, postStart)

	return iof.index
}

//----------

func (d *PosDrawer) BoundsPointOf(index int) image.Point {
	p := d.PointOf(index)
	return p.Sub(d.Offset()).Add(d.Bounds().Min)
}

func (d *PosDrawer) BoundsIndexOf(p image.Point) int {
	p2 := p.Sub(d.Bounds().Min).Add(d.Offset())
	return d.IndexOf(p2)
}

//----------

func (d *PosDrawer) BoundsAnnotationsIndexOf(p image.Point) (int, int, bool) {
	p2 := p.Sub(d.Bounds().Min).Add(d.Offset())
	return d.annotationsIndexOf(p2)
}

func (d *PosDrawer) annotationsIndexOf(p image.Point) (int, int, bool) {
	if !d.ready() {
		return 0, 0, false
	}

	if !d.Annotations.On() {
		return 0, 0, false
	}

	p2 := mathutil.PIntf2(p)
	aiof := MakeAnnotationsIndexOf(&d.Annotations, p2)
	exts2 := append(d.pexts, []Ext{&d.ee, &aiof}...)

	postStart := func() {
		d.pd.RestoreCloseToPoint(p2) // setups keepers exts
	}

	RunExts(d, &d.rr, exts2, postStart)

	if aiof.entryIndex < 0 {
		return 0, 0, false
	}

	return aiof.entryIndex, aiof.entryOffset, true
}

//----------

func (d *PosDrawer) SetBounds(r image.Rectangle) {
	if d.WrapLine.On() && r.Dx() != d.Bounds().Dx() {
		d.SetNeedMeasure(true)
	}
	d.DrawerCommon.SetBounds(r)
}
func (d *PosDrawer) SetBoundsSize(size image.Point) {
	b := d.Bounds()
	b.Max = b.Min.Add(size)
	d.SetBounds(b)
}

//----------

//func (d *DrawerCommon) readerLength() (int64, error) {
//	return d.reader.Seek(0, io.SeekEnd)
//}

//func (d *PosDrawer) drawSize_(index int64) mathutil.PointIntf {
//	_, err := d.Reader().Seek(index, io.SeekStart)
//	if err != nil {
//		return mathutil.PointIntf{}
//	}

//	//rr := MakeRuneReader(m.Face(), m.Reader())

//	return mathutil.PointIntf{}
//}

//----------

//func (d *PosDrawer) lengthSize(l int64) mathutil.PointIntf {
//	rs := d.uniformRuneSize()

//	w := mathutil.Intf1(d.bounds.Dx())

//	runesPerLine := int64(w / rs.X)
//	if runesPerLine <= 0 {
//		runesPerLine = 1
//	} else if runesPerLine > l {
//		runesPerLine = l
//	}

//	nlines := l / runesPerLine
//	if nlines <= 0 {
//		nlines = 1
//	}

//	u := mathutil.PIntf1(runesPerLine, nlines)
//	sizeX := rs.X.Mul(u.X)
//	sizeY := rs.Y.Mul(u.Y)

//	return mathutil.PointIntf{sizeX, sizeY}
//}

//func (d *PosDrawer) uniformRuneSize() mathutil.PointIntf {
//	metrics := d.face.Metrics()
//	lh := LineHeight(&metrics)
//	adv, ok := d.face.GlyphAdvance('W')
//	if !ok {
//		adv = lh
//	}
//	adv2 := mathutil.Intf3(adv)
//	lh2 := mathutil.Intf3(lh)
//	return mathutil.PointIntf{adv2, lh2}
//}
