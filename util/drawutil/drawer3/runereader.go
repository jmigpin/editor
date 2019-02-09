package drawer3

import (
	"image"
	"io"

	"github.com/jmigpin/editor/util/drawutil"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/mathutil"
	"golang.org/x/image/font"
)

type RuneReader struct {
	EExt

	Pen        mathutil.PointIntf // upper left corner (not at baseline)
	Ri         int
	Ru         rune
	PrevRu     rune
	Kern       mathutil.Intf
	Advance    mathutil.Intf
	LineHeight mathutil.Intf

	face    font.Face
	metrics font.Metrics
	reader  iorw.Reader

	riClone int
}

func (rr *RuneReader) Start(r *ExtRunner) {
	*rr = RuneReader{
		face:   r.D.Face(),
		reader: r.D.Reader(),
	}
	rr.Pen.X = mathutil.Intf1(r.D.FirstLineOffsetX())

	rr.metrics = rr.face.Metrics()
	lh := drawutil.LineHeight(&rr.metrics)
	rr.LineHeight = mathutil.Intf2(lh)
}

func (rr *RuneReader) Iterate(r *ExtRunner) {
	ru, size, err := rr.reader.ReadRuneAt(rr.Ri)
	if err != nil {
		// run exts at last advanced position for draw/selecting
		if err == io.EOF {
			rr.Iterate2(r, 0, 0)
		}
		r.Stop()
		return
	}
	_ = rr.Iterate2(r, ru, size)
}

func (rr *RuneReader) Iterate2(r *ExtRunner, ru rune, size int) bool {
	rr.Ru = ru

	// add/subtract kern with previous rune
	k := rr.face.Kern(rr.PrevRu, rr.Ru)
	rr.Kern = mathutil.Intf2(k)
	rr.Pen.X += rr.Kern

	// rune advance
	rr.Advance = rr.GlyphAdvance(rr.Ru)

	// tabulator
	if rr.Ru == '\t' {
		rr.Advance = rr.nextTabStopAdvance(rr.Pen.X, rr.Advance)
	}

	// run other exts
	if !r.NextExt() {
		return false
	}

	// advance for next rune
	rr.Ri += size
	rr.PrevRu = rr.Ru
	rr.Pen.X += rr.Advance

	return true
}

//----------

func (rr *RuneReader) PenBounds() mathutil.RectangleIntf {
	minX := rr.Pen.X
	minY := rr.Pen.Y
	maxX := minX + rr.Advance
	maxY := minY + rr.LineHeight
	min := mathutil.PointIntf{minX, minY}
	max := mathutil.PointIntf{maxX, maxY}
	return mathutil.RectangleIntf{min, max}
}

func (rr *RuneReader) OffsetPenBoundsRect(offset mathutil.PointIntf, pos image.Point) image.Rectangle {
	pb := rr.PenBounds()
	r := pb.Sub(offset)

	// expand min (use floor), and max (use ceil)
	rminX := r.Min.X.Floor()
	rminY := r.Min.Y.Floor()
	rmaxX := r.Max.X.Ceil()
	rmaxY := r.Max.Y.Ceil()

	r2 := image.Rect(rminX, rminY, rmaxX, rmaxY)

	return r2.Add(pos)
}

func (rr *RuneReader) OffsetPenPoint(offset mathutil.PointIntf, pos image.Point) image.Point {
	p := rr.Pen.Sub(offset)
	// pen is upper left corner, use floor
	p2 := image.Point{p.X.Floor(), p.Y.Floor()}
	return p2.Add(pos)
}

//----------

func (rr *RuneReader) PushRiClone() {
	rr.riClone++
}
func (rr *RuneReader) PopRiClone() {
	rr.riClone--
}
func (rr *RuneReader) RiClone() bool {
	return rr.riClone > 0
}

//----------

func (rr *RuneReader) GlyphAdvance(ru rune) mathutil.Intf {
	adv, ok := rr.face.GlyphAdvance(ru)
	if !ok {
		return 0
	}
	return mathutil.Intf2(adv)
}

//----------

func (rr *RuneReader) nextTabStopAdvance(penx, tadv mathutil.Intf) mathutil.Intf {
	x := penx + tadv
	n := int(x / tadv)
	nadv := mathutil.Intf(n) * tadv
	return nadv - penx
}

//----------

// Implements PosDataKeeper
func (rr *RuneReader) KeepPosData() interface{} {
	d := &RuneReaderData{
		Ri:     rr.Ri,
		PrevRu: rr.PrevRu,
		Pen:    rr.Pen,
	}
	return d
}

// Implements PosDataKeeper
func (rr *RuneReader) RestorePosData(data interface{}) {
	d := data.(*RuneReaderData)
	rr.Ri = d.Ri
	rr.PrevRu = d.PrevRu
	rr.Pen = d.Pen
}

//----------

type RuneReaderData struct {
	Ri     int
	PrevRu rune
	Pen    mathutil.PointIntf
}
