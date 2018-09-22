package drawer3

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/jmigpin/editor/util/drawutil"
	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/mathutil"
	"golang.org/x/image/math/fixed"
)

type DrawRune struct {
	EExt
	cc *CurColors

	// setup values
	img draw.Image

	// start values
	delay *DrawRuneDelay
}

func DrawRune1(cc *CurColors) DrawRune {
	return DrawRune{cc: cc}
}

func (dru *DrawRune) setup(img draw.Image) {
	dru.img = img
}

func (dru *DrawRune) Start(r *ExtRunner) {
	dru.delay = nil
}

func (dru *DrawRune) Iterate(r *ExtRunner) {
	// delay drawing by one rune to allow drawing the kern bg correctly
	// the last position is also drawn because the  runereader emits a final ru=0 at the end
	if dru.delay != nil {
		dru.draw(r, dru.delay)
	}
	dru.delay = NewDrawRuneDelay(r, dru)

	r.NextExt()
}

func (dru *DrawRune) draw(r *ExtRunner, delay *DrawRuneDelay) {
	// allow to skip draw with a rune 0
	if delay.ru == 0 {
		return
	}

	bl := fixed.Point26_6{X: 0, Y: drawutil.Baseline(&r.RR.metrics)}
	gr, mask, maskp, _, ok := r.RR.face.Glyph(bl, delay.ru)
	if !ok {
		return
	}

	// clip
	b := r.D.Bounds()
	gr = gr.Add(delay.penp)
	if gr.Min.X < b.Min.X {
		maskp.X += b.Min.X - gr.Min.X
	}
	if gr.Min.Y < b.Min.Y {
		maskp.Y += b.Min.Y - gr.Min.Y
	}
	gr = gr.Intersect(b)

	imageutil.DrawUniformMask(dru.img, &gr, delay.fg, mask, maskp, draw.Over)
}

//----------

type DrawRuneDelay struct {
	penp image.Point
	ru   rune
	fg   color.Color
}

func NewDrawRuneDelay(r *ExtRunner, dru *DrawRune) *DrawRuneDelay {
	offset := mathutil.PIntf2(r.D.Offset())
	pos := r.D.Bounds().Min
	return &DrawRuneDelay{
		penp: r.RR.OffsetPenPoint(offset, pos),
		ru:   r.RR.Ru,
		fg:   dru.cc.Fg,
	}
}
