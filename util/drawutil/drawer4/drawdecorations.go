package drawer4

import (
	"image"

	"github.com/jmigpin/editor/util/imageutil"
)

type DrawDecorations struct {
	d *Drawer
}

func (dd *DrawDecorations) Init() {}

func (dd *DrawDecorations) Iter() {
	dd.draw()
	if !dd.d.iterNext() {
		return
	}
}

func (dd *DrawDecorations) End() {}

func (dd *DrawDecorations) draw() {
	dec := dd.activeDecoration()
	if dec == nil || dec.Kind != DecorationHorizontalRule || dec.Fg == nil {
		return
	}

	r := dd.d.iters.runeR.penBoundsRect()
	b := dd.d.bounds
	th := dec.Thickness
	if th <= 0 {
		th = max(1, r.Dy()/12)
	}
	y := r.Min.Y - th/2

	adv := dd.d.iters.runeR.glyphAdvance('W').Ceil()
	if adv <= 0 {
		adv = 8
	}
	dashW := adv
	gapW := adv
	for x := b.Min.X; x < b.Max.X; x += dashW + gapW {
		r2 := image.Rect(x, y, min(x+dashW, b.Max.X), y+th)
		r2 = r2.Intersect(b)
		imageutil.FillRectangle(dd.d.st.drawR.img, r2, dec.Fg)
	}
}

func (dd *DrawDecorations) activeDecoration() *Decoration {
	if !dd.d.iters.runeR.isNormal() {
		return nil
	}
	ri := dd.d.st.runeR.ri
	for gi, g := range dd.d.Opt.Decorations.Groups {
		if g == nil || g.Off {
			continue
		}
		i := &dd.d.st.decorations.indexes[gi]
		for ; *i < len(g.Entries); *i++ {
			entry := g.Entries[*i]
			if entry == nil {
				continue
			}
			if !dd.d.iters.decorations.isValidLineStartOffset(entry.Offset) {
				continue
			}
			if entry.Offset < ri {
				continue
			}
			if entry.Offset > ri {
				break
			}
			return entry
		}
	}
	return nil
}
