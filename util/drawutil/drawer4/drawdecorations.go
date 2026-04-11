package drawer4

import (
	"image"
	"image/draw"

	"github.com/jmigpin/editor/util/imageutil"
)

type DrawDecorations struct {
	d *Drawer
}

func (dd *DrawDecorations) Init() {}

func (dd *DrawDecorations) Iter() {
	if !dd.d.iterNext() {
		return
	}
}

func (dd *DrawDecorations) End() {
	img := dd.d.st.drawR.img
	for _, g := range dd.d.Opt.Decorations.Groups {
		if g == nil || g.Off {
			continue
		}
		for _, dec := range g.Entries {
			if dec == nil {
				continue
			}
			dd.draw(img, dec)
		}
	}
}

func (dd *DrawDecorations) draw(img draw.Image, dec *Decoration) {
	if dec.Kind != DecorationHorizontalRule || dec.Fg == nil {
		return
	}

	p := dd.d.LocalPointOf(dec.Offset)
	b := dd.d.bounds
	th := dec.Thickness
	if th <= 0 {
		th = max(1, dd.d.fface.LineHeightInt()/12)
	}
	y := p.Y - th/2

	adv, ok := dd.d.fface.Face.GlyphAdvance('W')
	if !ok {
		adv = 0
	}
	dashW := adv.Ceil()
	if dashW <= 0 {
		dashW = 8
	}
	gapW := dashW
	for x := b.Min.X; x < b.Max.X; x += dashW + gapW {
		r2 := image.Rect(x, y, min(x+dashW, b.Max.X), y+th)
		r2 = r2.Intersect(b)
		imageutil.FillRectangle(img, r2, dec.Fg)
	}
}
