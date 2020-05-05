package drawer4

import (
	"github.com/jmigpin/editor/v2/util/mathutil"
)

func header1PenBounds(d *Drawer, offset int) (mathutil.RectangleIntf, bool) {
	d.st = State{}
	fnIter := FnIter{}
	iters := append(d.sIters(), &d.iters.earlyExit, &fnIter)
	d.loopInit(iters)
	d.header1()

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

type PenVisibility struct {
	not     bool // not visible
	full    bool // fully visible
	partial bool // partially visible
	top     bool // otherwise is bottom, valid in "full" and "partial"
}

func penVisibility(d *Drawer, offset int) *PenVisibility {
	v := &PenVisibility{}
	pb, ok := header1PenBounds(d, offset)
	if !ok {
		v.not = true
	} else {
		pr := pb.ToRectFloorCeil()
		// allow intersection of empty x in penbounds (case of eof)
		if pr.Dx() == 0 {
			pr.Max.X = pr.Min.X + 1
		}

		// consider previous/next lines (allows cursor up/down to move 1 line instead of jumping the view aligned to the center)
		b := d.bounds // copy
		b.Min.Y--
		b.Max.Y++

		ir := b.Intersect(pr)
		if ir.Empty() {
			v.not = true
		} else if ir == pr {
			v.full = true
		} else {
			v.partial = true
			if pr.Min.Y < b.Min.Y {
				v.top = true
			}
		}
	}
	return v
}
