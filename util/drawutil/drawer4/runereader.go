package drawer4

import (
	"image"
	"io"

	"github.com/jmigpin/editor/util/mathutil"
)

type RuneReader struct {
	d *Drawer
}

func (rr *RuneReader) Init() {
	rr.d.st.runeR.pen = rr.startingPen()
	rr.d.st.runeR.startRi = -1
}

func (rr *RuneReader) Iter() {
	// initialize start ri
	if rr.d.st.runeR.startRi == -1 {
		rr.d.st.runeR.startRi = rr.d.st.runeR.ri
	}

	ru, size, err := rr.d.reader.ReadRuneAt(rr.d.st.runeR.ri)
	if err != nil {
		// run last advanced position (draw/delayeddraw/selecting)
		if err == io.EOF {
			_ = rr.iter2(eofRune, 0)
		}
		rr.d.iterStop()
		return
	}
	_ = rr.iter2(ru, size)
}

func (rr *RuneReader) End() {}

//----------

func (rr *RuneReader) eof() bool {
	return rr.isNormal() && rr.d.st.runeR.ru == 0
}

func (rr *RuneReader) iter2(ru rune, size int) bool {
	st := &rr.d.st.runeR
	st.ru = ru

	// add/subtract kern with previous rune
	k := rr.d.fface.Face.Kern(st.prevRu, st.ru)
	st.kern = mathutil.Intf2(k)
	st.pen.X += st.kern

	// rune advance
	st.advance = rr.glyphAdvance(st.ru)

	// tabulator
	if st.ru == '\t' {
		st.advance = rr.nextTabStopAdvance(st.pen.X, st.advance)
	}

	if !rr.d.iterNext() {
		return false
	}

	// advance for next rune
	st.ri += size
	st.prevRu = st.ru
	st.pen.X += st.advance

	return true
}

//----------

func (rr *RuneReader) insertExtraString(s string) bool {
	rr.pushExtra()
	defer rr.popExtra()

	for _, ru := range s {
		if !rr.iter2(ru, len(string(ru))) {
			return false
		}
	}
	return true
}

//----------

func (rr *RuneReader) pushExtra() {
	rr.d.st.runeR.extra++
}
func (rr *RuneReader) popExtra() {
	rr.d.st.runeR.extra--
}
func (rr *RuneReader) isExtra() bool {
	return rr.d.st.runeR.extra > 0
}
func (rr *RuneReader) isNormal() bool {
	return !rr.isExtra()
}

//----------

func (rr *RuneReader) glyphAdvance(ru rune) mathutil.Intf {
	adv, ok := rr.d.fface.Face.GlyphAdvance(ru)
	if !ok {
		return 0
	}
	return mathutil.Intf2(adv)
}

func (rr *RuneReader) nextTabStopAdvance(penx, tadv mathutil.Intf) mathutil.Intf {
	px := penx - rr.startingPen().X
	x := px + tadv
	n := int(x / tadv)
	nadv := mathutil.Intf(n) * tadv
	return nadv - px
}

//----------

func (rr *RuneReader) penBounds() mathutil.RectangleIntf {
	st := &rr.d.st.runeR
	minX, minY := st.pen.X, st.pen.Y
	maxX, maxY := minX+st.advance, minY+rr.d.lineHeight
	min := mathutil.PointIntf{minX, minY}
	max := mathutil.PointIntf{maxX, maxY}
	return mathutil.RectangleIntf{min, max}
}

func (rr *RuneReader) penBoundsRect() image.Rectangle {
	pb := rr.penBounds()
	// expand min (use floor), and max (use ceil)
	return pb.ToRectFloorCeil()
}

//----------

func (rr *RuneReader) startingPen() mathutil.PointIntf {
	p := rr.d.bounds.Min
	p.X += rr.d.Opt.RuneReader.StartOffsetX
	if rr.d.st.runeR.ri == 0 {
		p.X += rr.d.firstLineOffsetX
	}
	return mathutil.PIntf2(p)
}

func (rr *RuneReader) maxX() mathutil.Intf {
	return mathutil.Intf1(rr.d.bounds.Max.X)
}

//----------

type runeType int

const (
	rtNormal runeType = iota
	rtBackground
	rtInserted
	//rtAnnotation
)
