package drawer4

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/jmigpin/editor/util/drawutil"
	"github.com/jmigpin/editor/util/drawutil/drawer3"
	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/mathutil"
	"golang.org/x/image/font"
)

// TODO:
/*
	time to load: due to hash compute from erow? needs to read whole file
*/

type Drawer struct {
	reader iout.Reader

	face             font.Face
	metrics          font.Metrics
	lineHeight       mathutil.Intf
	offset           image.Point
	bounds           image.Rectangle
	firstLineOffsetX int
	startOffsetX     int

	measureId   int
	needMeasure bool
	measurement image.Point

	st State

	iters struct {
		runeR              RuneReader
		runeO              RuneOffset
		measure            Measure
		drawR              DrawRune
		line               Line
		lineWrap           LineWrap
		indent             Indent
		earlyExit          EarlyExit
		curColors          CurColors
		bgFill             BgFill
		cursor             Cursor
		pointOf            PointOf
		indexOf            IndexOf
		colorize           Colorize
		annotations        Annotations
		annotationsIndexOf AnnotationsIndexOf
	}

	Opt struct {
		RuneOffset struct {
			On     bool
			offset int
			line2  struct {
				calc struct {
					offset    int
					measureId int
				}
				start, end int
			}
		}
		LineWrap struct {
			Fg, Bg color.Color
		}
		Cursor struct {
			On    bool
			index int
			Fg    color.Color
		}
		Colorize struct {
			Groups []*ColorizeGroup
		}
		Annotations struct {
			On       bool
			Fg, Bg   color.Color
			Selected struct {
				EntryIndex int
				Fg, Bg     color.Color
			}
			Entries []*drawer3.Annotation // must be ordered by offset
		}
		WordHighlight struct {
			On     bool
			Fg, Bg color.Color
			word   []byte
			Group  ColorizeGroup
		}
		ParenthesisHighlight struct {
			On     bool
			Fg, Bg color.Color
			Group  ColorizeGroup
		}
		SyntaxHighlight struct {
			Comment struct {
				Line struct {
					S      string
					Fg, Bg color.Color
				}
				Enclosed struct {
					S, E   string
					Fg, Bg color.Color
				}
			}
			String struct {
				Fg, Bg color.Color
			}
			Group ColorizeGroup
		}
	}
}

type State struct {
	iters []Iterator

	loop struct {
		i    int
		stop bool
	}

	runeR struct {
		ri            int
		ru, prevRu    rune
		pen           mathutil.PointIntf // upper left corner (not at baseline)
		kern, advance mathutil.Intf
		riExtra       int // extra rune, doesn't exist in original content
	}
	measure struct {
		penMax mathutil.PointIntf
	}
	drawR struct {
		img   draw.Image
		delay *DrawRuneDelay
	}
	line     struct{}
	lineWrap struct {
		maxX    mathutil.Intf
		wrapped bool
	}
	indent struct {
		notStartingSpaces bool
		indent            mathutil.Intf
	}
	earlyExit struct {
		maxY mathutil.Intf
	}
	curColors struct {
		startFg color.Color
		fg, bg  color.Color
	}
	bgFill struct{}
	cursor struct {
		delay *CursorDelay
	}
	pointOf struct {
		index int
		p     image.Point
	}
	indexOf struct {
		p     mathutil.PointIntf
		index int
	}
	colorize struct {
		indexes []int
	}
	annotations struct {
		cei    int // current entries index (to add to q)
		indexQ []int
	}
	annotationsIndexOf struct {
		p      mathutil.PointIntf
		eindex int
		offset int
		inside struct { // inside an annotation
			on      bool
			ei      int // entry index
			soffset int // start offset
		}
	}
}

type Iterator interface {
	Init()
	Iter()
	End()
}

//----------

func New() *Drawer {
	d := &Drawer{}
	d.startOffsetX = 1 // covers cursor pixel
	// iterators
	d.iters.runeR.d = d   // init
	d.iters.runeO.d = d   // init
	d.iters.measure.d = d // end
	d.iters.drawR.d = d
	d.iters.line.d = d
	d.iters.lineWrap.d = d // init
	d.iters.indent.d = d
	d.iters.earlyExit.d = d // init
	d.iters.curColors.d = d
	d.iters.bgFill.d = d
	d.iters.cursor.d = d
	d.iters.pointOf.d = d  // end
	d.iters.indexOf.d = d  // end
	d.iters.colorize.d = d // init
	d.iters.annotations.d = d
	d.iters.annotationsIndexOf.d = d
	return d
}

//----------

func (d *Drawer) SetReader(r iout.Reader) { d.reader = r }
func (d *Drawer) Reader() iout.Reader     { return d.reader }

func (d *Drawer) Offset() image.Point     { return d.offset }
func (d *Drawer) SetOffset(o image.Point) { /*d.offset = o*/ }

func (d *Drawer) Face() font.Face { return d.face }
func (d *Drawer) SetFace(f font.Face) {
	if f != d.face {
		d.face = f
		d.metrics = d.face.Metrics()
		lh := drawutil.LineHeight(&d.metrics)
		d.lineHeight = mathutil.Intf2(lh)
		d.SetNeedMeasure(true)
	}
}
func (d *Drawer) LineHeight() int {
	if d.face == nil {
		return 0
	}
	return d.lineHeight.Floor() // already ceiled at linheight, use floor
}

func (d *Drawer) Bounds() image.Rectangle { return d.bounds }
func (d *Drawer) SetBounds(r image.Rectangle) {
	if r.Dx() != d.Bounds().Dx() {
		d.SetNeedMeasure(true)
	}
	d.bounds = r
}
func (d *Drawer) SetBoundsSize(size image.Point) {
	b := d.Bounds()
	b.Max = b.Min.Add(size)
	d.SetBounds(b)
}

func (d *Drawer) NeedMeasure() bool     { return d.needMeasure }
func (d *Drawer) SetNeedMeasure(v bool) { d.needMeasure = v }

func (d *Drawer) FirstLineOffsetX() int { return d.firstLineOffsetX }
func (d *Drawer) SetFirstLineOffsetX(x int) {
	if x != d.firstLineOffsetX {
		d.firstLineOffsetX = x
		d.SetNeedMeasure(true)
	}
}

func (d *Drawer) BoundsPointOf(index int) image.Point {
	p := d.PointOf(index)
	return p.Sub(d.Offset()).Add(d.Bounds().Min)
}
func (d *Drawer) BoundsIndexOf(p image.Point) int {
	p2 := p.Sub(d.Bounds().Min).Add(d.Offset())
	return d.IndexOf(p2)
}

//----------

func (d *Drawer) RuneOffset() int     { return d.Opt.RuneOffset.offset }
func (d *Drawer) SetRuneOffset(v int) { d.Opt.RuneOffset.offset = v }

func (d *Drawer) SetCursorIndex(v int) {
	d.Opt.Cursor.index = v
	// TODO: on paste, the cursor index might not changed, but these need to be updated
	updateWordHighlightWord(d)
	updateParenthesisHighlight(d, 5000) // max distance to find close
}

//----------

func (d *Drawer) ready() bool {
	return !(d.face == nil || d.reader == nil || d.bounds == image.ZR)
}

//----------

func (d *Drawer) Measure() image.Point {
	if !d.ready() {
		return image.Point{}
	}
	if !d.needMeasure {
		return d.measurement
	}
	d.needMeasure = false
	d.measureId++
	if d.Opt.RuneOffset.On {
		d.measurement = d.measureForRuneOffset()
	} else {
		d.measurement = d.measurePixels()
	}
	return d.measurement
}

func (d *Drawer) measurePixels() image.Point {
	d.st = State{}
	iters := []Iterator{
		&d.iters.runeR,
		&d.iters.line,
		&d.iters.lineWrap,
		&d.iters.indent,
		&d.iters.measure,
	}
	d.loopInit(iters)
	d.loop()
	return d.st.measure.penMax.ToPointCeil()
}

func (d *Drawer) measureForRuneOffset() image.Point {
	return image.Point{1, d.reader.Len()}
}

//----------

func (d *Drawer) measureContent(offset, n int) mathutil.PointIntf {
	// keep/restore state to allow running inside an already called iteration
	st := d.st
	defer func() { d.st = st }()

	d.st = State{}
	iters := []Iterator{
		&d.iters.runeR,
		&d.iters.line,
		&d.iters.lineWrap,
		&d.iters.indent,
		&d.iters.pointOf,
	}
	d.loopInit(iters)
	d.st.runeR.ri = offset
	d.st.pointOf.index = offset + n
	d.loop()
	return d.iters.runeR.penBounds().Max
}

//----------

func (d *Drawer) Draw(img draw.Image, fg color.Color) {
	updateSyntaxHighlightOps(d, 5000)
	updateWordHighlightOps(d)

	d.st = State{}
	d.st.drawR.img = img
	d.st.curColors.startFg = fg
	iters := []Iterator{
		&d.iters.curColors,
		&d.iters.runeO,
		&d.iters.runeR,
		&d.iters.colorize,
		&d.iters.line,
		&d.iters.lineWrap,    // inserts extra runes
		&d.iters.indent,      // inserts extra runes
		&d.iters.earlyExit,   // after iters that change pen.Y
		&d.iters.annotations, // inserts extra runes
		&d.iters.bgFill,
		&d.iters.drawR,
		&d.iters.cursor,
	}
	d.loopInit(iters)
	d.loop()
}

//----------

func (d *Drawer) PointOf(index int) image.Point {
	if !d.ready() {
		return image.Point{}
	}
	d.st = State{}
	d.st.pointOf.index = index
	iters := []Iterator{
		&d.iters.runeO,
		&d.iters.runeR,
		&d.iters.line,
		&d.iters.lineWrap,
		&d.iters.indent,
		&d.iters.pointOf,
	}
	d.loopInit(iters)
	d.loop()
	return d.st.pointOf.p
}

//----------

func (d *Drawer) IndexOf(p image.Point) int {
	if !d.ready() {
		return 0
	}
	d.st = State{}
	d.st.indexOf.p = mathutil.PIntf2(p)
	iters := []Iterator{
		&d.iters.runeO,
		&d.iters.runeR,
		&d.iters.line,
		&d.iters.lineWrap,
		&d.iters.indent,
		&d.iters.indexOf,
	}
	d.loopInit(iters)
	d.loop()
	return d.st.indexOf.index
}

//----------

func (d *Drawer) AnnotationsIndexOf(p image.Point) (int, int, bool) {
	if !d.ready() {
		return 0, 0, false
	}
	d.st = State{}
	d.st.annotationsIndexOf.p = mathutil.PIntf2(p)
	iters := []Iterator{
		&d.iters.runeO,
		&d.iters.runeR,
		&d.iters.line,
		&d.iters.lineWrap,
		&d.iters.indent,
		&d.iters.annotations,
		&d.iters.annotationsIndexOf,
	}
	d.loopInit(iters)
	d.loop()

	st := &d.st.annotationsIndexOf
	if st.eindex < 0 {
		return 0, 0, false
	}
	return st.eindex, st.offset, true
}

func (d *Drawer) BoundsAnnotationsIndexOf(p image.Point) (int, int, bool) {
	p2 := p.Sub(d.Bounds().Min).Add(d.Offset())
	return d.AnnotationsIndexOf(p2)
}

//----------

func (d *Drawer) loopInit(iters []Iterator) {
	d.st.iters = iters
	for _, iter := range d.st.iters {
		iter.Init()
	}
}

func (d *Drawer) loop() {
	for !d.st.loop.stop { // loop for each rune
		d.st.loop.i = 0
		_ = d.iterNext()
	}
	for _, iter := range d.st.iters {
		iter.End()
	}
}

// To be called from extensions, inside the Iter() func.
func (d *Drawer) iterNext() bool {
	st := &d.st
	if st.loop.i < len(st.iters) {
		u := st.iters[st.loop.i]
		st.loop.i++
		u.Iter()
		st.loop.i--
	}
	return !st.loop.stop
}

func (d *Drawer) iterStop() {
	d.st.loop.stop = true
}

//----------

func (d *Drawer) ScrollableViewSize() image.Point {
	if d.Opt.RuneOffset.On {
		y := d.runeOffsetViewLen()
		return image.Point{1, y}
	}
	return d.bounds.Size()
}

//----------

func (d *Drawer) runeOffsetViewLen() int {
	if !d.ready() {
		return 0
	}
	if !d.Opt.RuneOffset.On {
		return 0
	}
	d.st = State{}
	iters := []Iterator{
		&d.iters.runeO,
		&d.iters.runeR,
		&d.iters.line,
		&d.iters.lineWrap,
		&d.iters.indent,
		&d.iters.earlyExit,
	}
	d.loopInit(iters)
	d.loop()
	return d.st.runeR.ri - d.Opt.RuneOffset.offset
}

//----------

func assignColor(dest *color.Color, src color.Color) {
	if src != nil {
		*dest = src
	}
}
