package drawer4

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/davecgh/go-spew/spew"
	"github.com/jmigpin/editor/util/drawutil"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/mathutil"
	"golang.org/x/image/font"
)

const (
	eofRune    = -1
	noDrawRune = -2
)

type Drawer struct {
	reader iorw.Reader

	face             font.Face
	metrics          font.Metrics
	lineHeight       mathutil.Intf
	offset           image.Point
	bounds           image.Rectangle
	firstLineOffsetX int
	fg               color.Color
	smoothScroll     bool

	iters struct {
		runeR              RuneReader // init
		measure            Measure    // end
		drawR              DrawRune
		line               Line
		lineWrap           LineWrap  // init, insert
		lineStart          LineStart // init
		indent             Indent    // insert
		earlyExit          EarlyExit
		curColors          CurColors
		bgFill             BgFill
		cursor             Cursor
		pointOf            PointOf     // end
		indexOf            IndexOf     // end
		colorize           Colorize    // init
		annotations        Annotations // insert
		annotationsIndexOf AnnotationsIndexOf
	}

	st State

	loopv struct {
		iters []Iterator
		i     int
		stop  bool
	}

	// internal opt data
	opt struct {
		measure struct {
			updated bool
			measure image.Point
		}
		runeO struct {
			offset int
		}
		cursor struct {
			offset int
		}
		wordH struct {
			word        []byte
			updatedWord bool
			updatedOps  bool
		}
		parenthesisH struct {
			updated bool
		}
		syntaxH struct {
			updated bool
		}
	}

	// external options
	Opt struct {
		RuneReader struct {
			StartOffsetX int
		}
		RuneOffset struct {
			On bool
		}
		LineWrap struct {
			Fg, Bg color.Color
		}
		Cursor struct {
			On bool
			Fg color.Color
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
			Entries []*Annotation // must be ordered by offset
		}
		WordHighlight struct {
			On     bool
			Fg, Bg color.Color
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

// State should not be stored/restored except in initializations.
// ex: runeR.extra and runeR.ru won't be correctly set if the iterators were stopped.
type State struct {
	runeR struct {
		ri            int
		ru, prevRu    rune
		pen           mathutil.PointIntf // upper left corner (not at baseline)
		kern, advance mathutil.Intf
		extra         int
		startRi       int
	}
	measure struct {
		penMax mathutil.PointIntf
	}
	drawR struct {
		img   draw.Image
		delay *DrawRuneDelay
	}
	line struct {
		lineStart bool
	}
	lineWrap struct {
		wrapRi       int
		preLineWrap  bool
		postLineWrap bool // post line wrap
	}
	lineStart struct {
		offset     int
		nLinesUp   int
		q          []int
		ri         int
		uppedLines int
		reader     iorw.Reader // limited reader
	}
	indent struct {
		notStartingSpaces bool
		indent            mathutil.Intf
	}
	earlyExit struct {
		extraLine bool
	}
	curColors struct {
		fg, bg color.Color
		lineBg color.Color
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

func (st State) Dump() {
	st.drawR.img = nil
	spew.Dump(st)
}

//----------

func New() *Drawer {
	d := &Drawer{}
	d.smoothScroll = true

	// iterators
	d.iters.runeR.d = d
	d.iters.measure.d = d
	d.iters.drawR.d = d
	d.iters.line.d = d
	d.iters.lineWrap.d = d
	d.iters.lineStart.d = d
	d.iters.indent.d = d
	d.iters.earlyExit.d = d
	d.iters.curColors.d = d
	d.iters.bgFill.d = d
	d.iters.cursor.d = d
	d.iters.pointOf.d = d
	d.iters.indexOf.d = d
	d.iters.colorize.d = d
	d.iters.annotations.d = d
	d.iters.annotationsIndexOf.d = d
	return d
}

//----------

func (d *Drawer) SetReader(r iorw.Reader) { d.reader = r }

func (d *Drawer) Reader() iorw.Reader { return d.reader }

//----------

var limitedReaderPadding = 3000

func (d *Drawer) limitedReaderPad(offset int) iorw.Reader {
	return iorw.NewLimitedReader(d.reader, offset, offset, limitedReaderPadding)
}

func (d *Drawer) limitedReaderPadSpace(offset int) iorw.Reader {
	// adjust the padding to avoid immediate flicker for x chars for the case of long lines
	u := offset - limitedReaderPadding
	diff := 1000 - (u % 1000)
	pad := limitedReaderPadding - diff
	return iorw.NewLimitedReader(d.reader, offset, offset, pad)
}

//----------

func (d *Drawer) ContentChanged() {
	d.opt.measure.updated = false
	d.opt.syntaxH.updated = false
	d.opt.wordH.updatedOps = false
	d.opt.parenthesisH.updated = false
}

//----------

func (d *Drawer) Face() font.Face { return d.face }
func (d *Drawer) SetFace(f font.Face) {
	if f == d.face {
		return
	}
	d.face = f
	d.metrics = d.face.Metrics()
	lh := drawutil.LineHeight(&d.metrics)
	d.lineHeight = mathutil.Intf2(lh)

	d.opt.measure.updated = false
}

func (d *Drawer) LineHeight() int {
	if d.face == nil {
		return 0
	}
	return d.lineHeight.Floor() // already ceiled at linheight, use floor
}

func (d *Drawer) SetFg(fg color.Color) { d.fg = fg }

//----------

func (d *Drawer) FirstLineOffsetX() int { return d.firstLineOffsetX }
func (d *Drawer) SetFirstLineOffsetX(x int) {
	if x != d.firstLineOffsetX {
		d.firstLineOffsetX = x
		d.opt.measure.updated = false
	}
}

//----------

func (d *Drawer) Bounds() image.Rectangle { return d.bounds }
func (d *Drawer) SetBounds(r image.Rectangle) {
	if r.Size() != d.bounds.Size() {
		d.opt.measure.updated = false
		d.opt.syntaxH.updated = false
		d.opt.wordH.updatedOps = false
		d.opt.parenthesisH.updated = false
	}
	d.bounds = r // always update value (can change min)
}

//----------

func (d *Drawer) RuneOffset() int {
	return d.opt.runeO.offset
}
func (d *Drawer) SetRuneOffset(v int) {
	d.opt.runeO.offset = v

	d.opt.syntaxH.updated = false
	d.opt.wordH.updatedOps = false
	d.opt.parenthesisH.updated = false
}

//----------

func (d *Drawer) SetCursorOffset(v int) {
	d.opt.cursor.offset = v

	d.opt.wordH.updatedWord = false
	d.opt.wordH.updatedOps = false
	d.opt.parenthesisH.updated = false
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
	if d.opt.measure.updated {
		return d.opt.measure.measure
	}
	d.opt.measure.updated = true
	d.opt.measure.measure = d.measure2()
	return d.opt.measure.measure
}

func (d *Drawer) measure2() image.Point {
	if d.Opt.RuneOffset.On {
		return d.bounds.Size()
	}
	return d.measurePixels()
}

func (d *Drawer) measurePixels() image.Point {
	d.st = State{}
	iters := []Iterator{
		&d.iters.runeR,
		&d.iters.line,
		&d.iters.lineWrap,
		&d.iters.indent,
		&d.iters.earlyExit,
		&d.iters.measure,
	}
	d.loopInit(iters)
	d.loop()
	// remove bounds min and return only the measure
	p := d.st.measure.penMax.ToPointCeil()
	m := p.Sub(d.bounds.Min)
	return m
}

//----------

func (d *Drawer) Draw(img draw.Image) {
	updateSyntaxHighlightOps(d)
	updateWordHighlightWord(d)
	updateWordHighlightOps(d)
	updateParenthesisHighlight(d)

	d.st = State{}
	iters := []Iterator{
		&d.iters.runeR,
		&d.iters.curColors,
		&d.iters.colorize,
		&d.iters.line,
		&d.iters.lineWrap,
		&d.iters.indent,
		&d.iters.earlyExit,   // after iters that change pen.Y
		&d.iters.annotations, // after iters that change the line
		&d.iters.bgFill,
		&d.iters.drawR,
		&d.iters.cursor,
	}
	d.loopInit(iters)
	d.header0()
	d.st.drawR.img = img
	d.loop()
}

//----------

func (d *Drawer) LocalPointOf(index int) image.Point {
	if !d.ready() {
		return image.Point{}
	}
	d.st = State{}
	d.st.pointOf.index = index
	iters := []Iterator{
		&d.iters.runeR,
		&d.iters.line,
		&d.iters.lineWrap,
		&d.iters.indent,
		&d.iters.earlyExit,
		&d.iters.pointOf,
	}
	d.loopInit(iters)
	d.header1()
	d.loop()
	return d.st.pointOf.p
}

//----------

func (d *Drawer) LocalIndexOf(p image.Point) int {
	if !d.ready() {
		return 0
	}
	d.st = State{}
	d.st.indexOf.p = mathutil.PIntf2(p)
	iters := []Iterator{
		&d.iters.runeR,
		&d.iters.line,
		&d.iters.lineWrap,
		&d.iters.indent,
		&d.iters.earlyExit,
		&d.iters.indexOf,
	}
	d.loopInit(iters)
	d.header1()
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
		&d.iters.runeR,
		&d.iters.line,
		&d.iters.lineWrap,
		&d.iters.indent,
		&d.iters.annotations,
		&d.iters.annotationsIndexOf,
	}
	d.loopInit(iters)
	d.header0()
	d.loop()

	st := &d.st.annotationsIndexOf
	if st.eindex < 0 {
		return 0, 0, false
	}
	return st.eindex, st.offset, true
}

//----------

func (d *Drawer) loopInit(iters []Iterator) {
	l := &d.loopv
	l.iters = iters
	for _, iter := range l.iters {
		iter.Init()
	}
}

func (d *Drawer) loop() {
	l := &d.loopv
	l.stop = false
	for !l.stop { // loop for each rune
		l.i = 0
		_ = d.iterNext()
	}
	for _, iter := range l.iters {
		iter.End()
	}
}

// To be called from iterators, inside the Iter() func.
func (d *Drawer) iterNext() bool {
	l := &d.loopv
	if !l.stop && l.i < len(l.iters) {
		u := l.iters[l.i]
		l.i++
		u.Iter()
		l.i--
	}
	return !l.stop
}

func (d *Drawer) iterStop() {
	d.loopv.stop = true
}

func (d *Drawer) iterNextExtra() bool {
	d.iters.runeR.pushExtra()
	defer d.iters.runeR.popExtra()
	return d.iterNext()
}

//----------

func (d *Drawer) visibleLen() (int, int, int, int) {
	d.st = State{}
	iters := append(d.sIters(), &d.iters.earlyExit)
	d.loopInit(iters)
	d.header0()
	startRi := d.st.runeR.ri
	d.loop()

	// from the line start
	drawOffset := startRi
	drawLen := d.st.runeR.ri - drawOffset
	// from the current offset
	offset := d.opt.runeO.offset
	offsetLen := d.st.runeR.ri - offset

	return drawOffset, drawLen, offset, offsetLen
}

//----------

func (d *Drawer) ScrollOffset() image.Point {
	return image.Point{0, d.RuneOffset()}
}
func (d *Drawer) SetScrollOffset(o image.Point) {
	d.SetRuneOffset(o.Y)
}

func (d *Drawer) ScrollSize() image.Point {
	return image.Point{0, d.reader.Len()}
}

func (d *Drawer) ScrollViewSize() image.Point {
	nlines := d.boundsNLines()
	n := d.scrollSizeY(nlines, false) // n runes
	return image.Point{0, n}
}

//----------

func (d *Drawer) ScrollPageSizeY(up bool) int {
	nlines := d.boundsNLines()
	return d.scrollSizeY(nlines, up)
}

//----------

func (d *Drawer) ScrollWheelSizeY(up bool) int {
	nlines := d.boundsNLines()

	// limit nlines
	nlines /= 4
	if nlines < 1 {
		nlines = 1
	} else if nlines > 4 {
		nlines = 4
	}

	return d.scrollSizeY(nlines, up)
}

//----------

// integer lines
func (d *Drawer) boundsNLines() int {
	dy := mathutil.Intf1(d.bounds.Dy())
	nlines := dy.Floor() / d.lineHeight.Floor()
	return nlines
}

//----------

func (d *Drawer) scrollSizeY(nlines int, up bool) int {
	if up {
		o := d.scrollSizeYUp(nlines)
		return -(d.opt.runeO.offset - o)
	} else {
		o := d.scrollSizeYDown(nlines)
		return o - d.opt.runeO.offset
	}
}

//----------

func (d *Drawer) scrollSizeYUp(nlines int) int {
	return d.wlineStartIndex(true, d.opt.runeO.offset, nlines, nil)
}
func (d *Drawer) scrollSizeYDown(nlines int) int {
	return d.wlineStartIndexDown(d.opt.runeO.offset, nlines)
}

//----------

func (d *Drawer) RangeVisible(offset, length int) bool {
	_, v1 := header1Visibility(d, offset)
	_, v2 := header1Visibility(d, offset+length)
	for _, v := range []Visibility{v1, v2} {
		switch v {
		case fullyVisible, topPartVisible, bottomPartVisible:
			return true
		}
	}
	return false
}

//----------

func (d *Drawer) RangeVisibleOffset(offset, length int) int {
	_, v1 := header1Visibility(d, offset)
	//_, v2 := header1Visibility(d, offset+length)
	switch v1 {
	case fullyVisible:
		return mathutil.Smallest(d.opt.runeO.offset, d.reader.Len())
	case notVisible:
		return d.visibleAtCenter(offset, length)
	case topPartVisible, topNotVisible:
		return d.visibleAtTop(offset)
	case bottomPartVisible, bottomNotVisible:
		return d.visibleAtBottom(offset, length)
	}
	return d.visibleAtCenter(offset, length)
}

func (d *Drawer) visibleAtTop(offset int) int {
	return d.wlineStartIndex(true, offset, 0, nil)
}
func (d *Drawer) visibleAtBottom(offset, length int) int {
	nlines := d.rangeNLines(offset, length)
	bnlines := d.boundsNLines()
	u := bnlines - nlines
	if u < 0 {
		u = 0
	}
	return d.wlineStartIndex(true, offset, u, nil)
}

func (d *Drawer) visibleAtCenter(offset, length int) int {
	// detect nlines
	nlines := d.rangeNLines(offset, length)

	// centered
	bnlines := d.boundsNLines()
	if nlines >= bnlines {
		nlines = 0 // top
	} else {
		nlines = (bnlines - nlines) / 2
	}

	return d.wlineStartIndex(true, offset, nlines, nil)
}

//----------

func (d *Drawer) rangeNLines(offset, length int) int {
	pr1, pr2, ok := d.wlineRangePenBounds(offset, length)
	if ok {
		u := ((pr2.Min.Y - pr1.Min.Y) / d.lineHeight).Floor() + 1
		if u > 1 {
			return u
		}
	}

	return 1
}

func (d *Drawer) wlineRangePenBounds(offset, length int) (_, _ mathutil.RectangleIntf, _ bool) {
	var pr1, pr2 mathutil.RectangleIntf
	var ok1, ok2 bool
	d.wlineStartLoopFn(true, offset, 0,
		func() {
			ok1 = true
			pr1 = d.iters.runeR.penBounds()
		},
		func() {
			if d.st.runeR.ri == offset+length {
				ok2 = true
				pr2 = d.iters.runeR.penBounds()
				d.iterStop()
				return
			}
			if !d.iterNext() {
				return
			}
		})
	return pr1, pr2, ok1 && ok2
}

//----------

func (d *Drawer) wlineStartIndexDown(offset, nlinesDown int) int {
	count := 0
	startRi := 0
	d.wlineStartLoopFn(true, offset, 0,
		func() {
			startRi = d.st.runeR.ri
			if nlinesDown == 0 {
				d.iterStop()
			}
		},
		func() {
			if d.st.line.lineStart || d.st.lineWrap.postLineWrap {
				if d.st.runeR.ri != startRi { // bypass ri at line start
					count++
					if count >= nlinesDown {
						d.iterStop()
						return
					}
				}
			}
			if !d.iterNext() {
				return
			}
		})
	return d.st.runeR.ri
}

//----------

func (d *Drawer) header0() {
	_ = d.header(d.opt.runeO.offset, 0)
}

func (d *Drawer) header1() {
	d.st.earlyExit.extraLine = true       // extra line at bottom
	ul := d.header(d.opt.runeO.offset, 1) // extra line at top
	if ul > 0 {
		d.st.runeR.pen.Y -= d.lineHeight * mathutil.Intf(ul)
	}
}

//----------

func (d *Drawer) header(offset, nLinesUp int) int {
	// smooth scrolling
	adjustPenY := mathutil.Intf(0)
	if d.Opt.RuneOffset.On && d.smoothScroll {
		adjustPenY += d.smoothScrolling(offset)
	}

	// iterate to the wline start
	st1 := d.st // keep initialized state to refer to pen difference
	uppedLines := d.wlineStartState(false, offset, nLinesUp)
	adjustPenY += d.st.runeR.pen.Y - st1.runeR.pen.Y
	d.st.runeR.pen.Y -= adjustPenY

	return uppedLines
}

func (d *Drawer) smoothScrolling(offset int) mathutil.Intf {
	// keep/restore state to avoid interfering with other running iterations
	st := d.st
	defer func() { d.st = st }()

	s, e := d.wlineStartEnd(offset)
	t := e - s
	if t <= 0 {
		return 0
	}
	k := offset - s
	perc := float64(k) / float64(t)
	return mathutil.Intf(int64(float64(d.lineHeight) * perc))
}

func (d *Drawer) wlineStartEnd(offset int) (int, int) {
	s, e := 0, 0
	d.wlineStartLoopFn(true, offset, 0,
		func() {
			s = d.st.runeR.ri
		},
		func() {
			// the line will eventually wrap (finite screen size), no earlyexit testing
			if d.st.line.lineStart || d.st.lineWrap.postLineWrap {
				if d.st.runeR.ri > offset {
					e = d.st.runeR.ri
					d.iterStop()
					return
				}
			}
			if !d.iterNext() {
				return
			}
		})
	if e == 0 {
		e = d.st.runeR.ri
	}
	return s, e
}

//----------

func (d *Drawer) wlineStartLoopFn(clearState bool, offset, nLinesUp int, fnInit func(), fn func()) {
	// keep/restore iters
	iters := d.loopv.iters
	defer func() { d.loopv.iters = iters }()

	fnIter := FnIter{fn: fn}
	d.loopv.iters = append(d.sIters(), &fnIter)
	d.wlineStartState(clearState, offset, nLinesUp)
	fnInit()
	d.loop()
}

//----------

// Leaves the state at line start
func (d *Drawer) wlineStartState(clearState bool, offset, nLinesUp int) int {
	// keep/restore iters
	iters := d.loopv.iters
	defer func() { d.loopv.iters = iters }()

	// set limited reading here to have common limits on the next two calls
	//var rd iorw.Reader
	//rd := d.limitedReaderPad(offset)
	rd := d.limitedReaderPadSpace(offset)

	// find start (state will reach offset)
	cp := d.st // keep state
	k := d.wlineStartIndex(clearState, offset, nLinesUp, rd)
	uppedLines := d.st.lineStart.uppedLines

	// leave state at line start instead of offset
	d.st = cp // restore state
	_ = d.wlineStartIndex(clearState, k, 0, rd)

	return uppedLines
}

//----------

func (d *Drawer) wlineStartIndex(clearState bool, offset, nLinesUp int, rd iorw.Reader) int {
	if clearState {
		d.st = State{}
	}
	d.st.lineStart.offset = offset
	d.st.lineStart.nLinesUp = nLinesUp
	d.st.lineStart.reader = rd
	iters := append(d.sIters(), &d.iters.lineStart)
	d.loopInit(iters)
	d.loop()
	return d.st.lineStart.ri
}

//----------

// structure iterators
func (d *Drawer) sIters() []Iterator {
	return []Iterator{
		&d.iters.runeR,
		&d.iters.line,
		&d.iters.lineWrap,
		&d.iters.indent,
	}
}

//----------

type Iterator interface {
	Init()
	Iter()
	End()
}

//----------

type FnIter struct {
	fn func()
}

func (it *FnIter) Init() {}
func (it *FnIter) Iter() { it.fn() }
func (it *FnIter) End()  {}

//----------

func assignColor(dest *color.Color, src color.Color) {
	if src != nil {
		*dest = src
	}
}
