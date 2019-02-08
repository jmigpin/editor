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
	st := &rr.d.st.runeR
	st.pen.X = rr.startX()
}

func (rr *RuneReader) Iter() {
	ru, size, err := rr.d.reader.ReadRuneAt(rr.d.st.runeR.ri)
	if err != nil {
		// run last advanced position (draw/delayeddraw/selecting)
		if err == io.EOF {
			_ = rr.iter2(0, 0)
		}
		rr.d.iterStop()
		return
	}
	_ = rr.iter2(ru, size)
}

func (rr *RuneReader) End() {}

//----------

func (rr *RuneReader) iter2(ru rune, size int) bool {
	st := &rr.d.st.runeR
	st.ru = ru

	// add/subtract kern with previous rune
	k := rr.d.face.Kern(st.prevRu, st.ru)
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
	rr.pushRiExtra()
	defer rr.popRiExtra()
	for _, ru := range s {
		if !rr.iter2(ru, len(string(ru))) {
			return false
		}
	}
	return true
}

//----------

func (rr *RuneReader) pushRiExtra() {
	rr.d.st.runeR.riExtra++
}
func (rr *RuneReader) popRiExtra() {
	rr.d.st.runeR.riExtra--
}
func (rr *RuneReader) isRiExtra() bool {
	return rr.d.st.runeR.riExtra > 0
}

//----------

func (rr *RuneReader) glyphAdvance(ru rune) mathutil.Intf {
	adv, ok := rr.d.face.GlyphAdvance(ru)
	if !ok {
		return 0
	}
	return mathutil.Intf2(adv)
}

func (rr *RuneReader) nextTabStopAdvance(penx, tadv mathutil.Intf) mathutil.Intf {
	x := penx + tadv
	n := int(x / tadv)
	nadv := mathutil.Intf(n) * tadv
	return nadv - penx
}

//----------

func (rr *RuneReader) penBounds() mathutil.RectangleIntf {
	st := rr.d.st.runeR
	minX, minY := st.pen.X, st.pen.Y
	maxX, maxY := minX+st.advance, minY+rr.d.lineHeight
	min := mathutil.PointIntf{minX, minY}
	max := mathutil.PointIntf{maxX, maxY}
	return mathutil.RectangleIntf{min, max}
}

func (rr *RuneReader) offsetPenBoundsRect(offset mathutil.PointIntf, pos image.Point) image.Rectangle {
	pb := rr.penBounds()
	r := pb.Sub(offset)

	// expand min (use floor), and max (use ceil)
	rminX := r.Min.X.Floor()
	rminY := r.Min.Y.Floor()
	rmaxX := r.Max.X.Ceil()
	rmaxY := r.Max.Y.Ceil()

	r2 := image.Rect(rminX, rminY, rmaxX, rmaxY)

	return r2.Add(pos)
}

func (rr *RuneReader) offsetPenPoint(offset mathutil.PointIntf, pos image.Point) image.Point {
	p := rr.d.st.runeR.pen.Sub(offset)
	// pen is upper left corner, use floor
	p2 := image.Point{p.X.Floor(), p.Y.Floor()}
	return p2.Add(pos)
}

//----------

func (rr *RuneReader) startX() mathutil.Intf {
	v := rr.d.startOffsetX
	if rr.d.st.runeR.ri == 0 {
		v += rr.d.firstLineOffsetX
	}
	return mathutil.Intf1(v)
}

func (rr *RuneReader) maxX() mathutil.Intf {
	v := rr.d.Offset().X + rr.d.Bounds().Dx()
	return mathutil.Intf1(v)
}
