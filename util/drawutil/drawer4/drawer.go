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
		lineWrap           LineWrap // init, insert
		indent             Indent   // insert
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
	indent struct {
		notStartingSpaces bool
		indent            mathutil.Intf
	}
	earlyExit struct {
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
func (d *Drawer) Reader() iorw.Reader     { return d.reader }
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
	d.st.drawR.img = img
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
	d.header0(iters)
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
		&d.iters.runeR,
		&d.iters.line,
		&d.iters.lineWrap,
		&d.iters.indent,
		&d.iters.earlyExit,
		&d.iters.pointOf,
	}
	d.loopInit(iters)
	d.header0(iters)
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
		&d.iters.runeR,
		&d.iters.line,
		&d.iters.lineWrap,
		&d.iters.indent,
		&d.iters.indexOf,
	}
	d.loopInit(iters)
	d.header0(iters)
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
	d.header0(iters)
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
	d.header0(iters)
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
		o := d.scrollWheelYUp(nlines)
		return -(d.opt.runeO.offset - o)
	} else {
		o := d.scrollWheelYDown(nlines)
		return o - d.opt.runeO.offset
	}
}

//----------

func (d *Drawer) scrollWheelYUp(nlines int) int {
	return d.wlineStartIndex(true, d.opt.runeO.offset, nlines)
}
func (d *Drawer) scrollWheelYDown(nlines int) int {
	return d.wlineStartIndexDown(d.opt.runeO.offset, nlines)
}

//----------

func (d *Drawer) RangeVisible(offset, length int) bool {
	_, v1 := d.offsetVisibility(offset)
	_, v2 := d.offsetVisibility(offset + length)
	return v1+v2 > 0
}

// 0=not, 1=partial, 2=full
func (d *Drawer) offsetVisibility(offset int) (image.Rectangle, int) {
	pb, ok := d.visiblePenBounds(offset)
	// not visible
	if !ok {
		return image.Rectangle{}, 0
	}
	pr := pb.ToRectFloorCeil()
	ir := d.bounds.Intersect(pr)
	if ir.Empty() {
		return image.Rectangle{}, 0
	}
	// partially visible
	if ir != pr {
		return pr, 1
	}
	// fully visible
	return pr, 2
}

func (d *Drawer) visiblePenBounds(offset int) (mathutil.RectangleIntf, bool) {
	d.st = State{}
	fnIter := FnIter{}
	iters := append(d.sIters(), &d.iters.earlyExit, &fnIter)
	d.loopInit(iters)
	d.header0(iters)

	found := false
	pen := mathutil.RectangleIntf{}
	fnIter.fn = func() {
		if d.iters.runeR.isNormal() {
			if d.st.runeR.ri >= offset {
				if d.st.runeR.ri == offset {
					found = true
					pen = d.iters.runeR.penBounds()
				}
				d.iterStop()
				return
			}
		}
		if !d.iterNext() {
			return
		}
	}

	d.loop()

	return pen, found
}

//----------

func (d *Drawer) RangeVisibleOffset(offset, length int) int {
	pr1, v1 := d.offsetVisibility(offset)
	_, v2 := d.offsetVisibility(offset + length)
	// not visible
	if v1+v2 == 0 {
		// centered
		return d.RangeVisibleOffsetCentered(offset, length)
	}
	// fully visible
	if v1+v2 == 4 {
		// do nothing
		return d.opt.runeO.offset
	}
	// partial: top
	top := false
	if v1 == 0 {
		// v2 is partial, align to top
		top = true
	} else if v1 == 1 {
		u := d.bounds
		u.Max.Y = u.Min.Y + 1
		if !u.Intersect(pr1).Empty() {
			// v1 is partial at top
			top = true
		}
	}
	if top {
		return d.wlineStartIndex(true, offset, 0)
	}
	// partial: bottom
	// detect nlines
	nlines := d.rangeNLines(offset, length)
	bnlines := d.boundsNLines()
	u := bnlines - nlines
	if u < 0 {
		u = 0
	}
	return d.wlineStartIndex(true, offset, u)
}

func (d *Drawer) rangeNLines(offset, length int) int {
	pr1, pr2, ok := d.wlineRangePenBounds(offset, length)
	if ok {
		u := int((pr2.Min.Y-pr1.Min.Y)/d.lineHeight) + 1
		if u > 1 {
			return u
		}
	}

	return 1
}

func (d *Drawer) wlineRangePenBounds(offset, length int) (_, _ mathutil.RectangleIntf, _ bool) {
	var pr1, pr2 mathutil.RectangleIntf
	var ok1, ok2 bool
	d.sLoopWLineStart(true, offset, 0,
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

func (d *Drawer) RangeVisibleOffsetCentered(offset, length int) int {
	// detect nlines
	nlines := d.rangeNLines(offset, length)

	// centered
	bnlines := d.boundsNLines()
	if nlines >= bnlines {
		nlines = 0 // top
	} else {
		nlines = (bnlines - nlines) / 2
	}

	return d.wlineStartIndex(true, offset, nlines)
}

//----------

func (d *Drawer) wlineStartIndexDown(offset, nlinesDown int) int {
	count := 0
	startRi := 0
	d.sLoopWLineStart(true, offset, 0,
		func() {
			startRi = d.st.runeR.ri
			if nlinesDown == 0 {
				d.iterStop()
			}
		},
		func() {
			if d.st.runeR.ri != startRi { // bypass ri at line start
				if d.st.line.lineStart || d.st.lineWrap.postLineWrap {
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

func (d *Drawer) header0(origIters []Iterator) {
	d.header(origIters, d.opt.runeO.offset)
}

func (d *Drawer) header(origIters []Iterator, offset int) {
	// smooth scrolling
	adjustPenY := mathutil.Intf(0)
	if d.Opt.RuneOffset.On && d.smoothScroll {
		adjustPenY += d.smoothScrolling(offset)
	}

	// iterate to the wline start
	st1 := d.st // keep initialized state to refer to pen difference
	d.wlineStartState(false, offset, 0)
	adjustPenY += d.st.runeR.pen.Y - st1.runeR.pen.Y
	d.st.runeR.pen.Y -= adjustPenY

	d.loopv.iters = origIters // restore original iterators
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
	d.sLoopWLineStart(true, offset, 0,
		func() {
			s = d.st.runeR.ri
		},
		func() {
			// since the line will eventually wrap (finite screen size) there is not need for early exit testing
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

//func (d *Drawer) sLoopHeader(offset int, fnInit func(), fn func()) {
//	d.sLoop(true,
//		func() {
//			d.header(d.loopv.iters, offset)
//			fnInit()
//		},
//		fn)
//}

func (d *Drawer) sLoopWLineStart(init bool, offset, nlinesUp int, fnInit func(), fn func()) {
	d.wlineStartState(init, offset, nlinesUp)
	d.sLoop(false, fnInit, fn)
}

//----------

// Leaves the state at line start
func (d *Drawer) wlineStartState(init bool, offset, nlinesUp int) {
	cp := d.st // keep copy of initialized state
	k := d.sLoopWLineStartIndex(init, offset, nlinesUp)
	// leave state at line start instead of offset
	d.st = cp // restore state
	_ = d.sLoopWLineStartIndex(init, k, 0)
}

//----------

func (d *Drawer) wlineStartIndex(init bool, offset, nlinesUp int) int {
	return d.sLoopWLineStartIndex(init, offset, nlinesUp)
}

// Leaves the state at offset
func (d *Drawer) sLoopWLineStartIndex(init bool, offset, nlinesUp int) int {
	q := []int{}
	d.sLoopLineStart(init, offset, nlinesUp,
		func() {
			// worst case line start, ok if it is pushed twice into the q
			q = append(q, d.st.runeR.ri)
		},
		func() {
			if d.st.line.lineStart || d.st.lineWrap.postLineWrap {
				q = append(q, d.st.runeR.ri)
			}
			if !d.st.lineWrap.preLineWrap { // don't stop before lineStart
				if d.st.runeR.ri >= offset {
					d.iterStop()
					return
				}
			}
			if !d.iterNext() {
				return
			}
		})
	// count lines back
	if nlinesUp >= len(q) {
		nlinesUp = len(q) - 1
	}
	return q[len(q)-1-nlinesUp]
}

//----------

func (d *Drawer) sLoopLineStart(init bool, offset, nlinesUp int, fnInit func(), fn func()) {
	d.sLoop(init,
		func() {
			// start iterating at the start of the content line
			d.st.runeR.ri = d.lineStartIndex(offset, nlinesUp)

			fnInit()
		},
		fn)
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

// structure iterators loop
func (d *Drawer) sLoop(init bool, fnInit func(), fn func()) {
	if init {
		d.st = State{}
	}
	fnIter := FnIter{fn: fn}
	iters := append(d.sIters(), &fnIter)
	if init {
		d.loopInit(iters)
	} else {
		d.loopv.iters = iters
	}
	fnInit()
	d.loop()
}

//----------

func (d *Drawer) lineStartIndex(offset, nlinesUp int) int {
	w := d.linesStartIndexes(offset, nlinesUp)

	// read error case
	if len(w) == 0 {
		return offset
	}

	if nlinesUp >= len(w) {
		nlinesUp = len(w) - 1
	}
	return w[nlinesUp]
}

func (d *Drawer) linesStartIndexes(offset, nlinesUp int) []int {
	w := []int{}
	for i := 0; i <= nlinesUp; i++ {
		k, err := iorw.LineStartIndex(d.reader, offset)
		if err != nil {
			if err == iorw.ErrLimitReached {
				// consider the limit as the line start
				w = append(w, k)
			}
			break
		}
		w = append(w, k)
		offset = k - 1
	}
	return w
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
