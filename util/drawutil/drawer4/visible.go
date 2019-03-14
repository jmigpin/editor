package drawer4

import (
	"image"

	"github.com/jmigpin/editor/util/mathutil"
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

type Visibility int

const (
	notVisible Visibility = iota
	fullyVisible
	topPartVisible
	topNotVisible
	bottomPartVisible
	bottomNotVisible
)

func header1Visibility(d *Drawer, offset int) (image.Rectangle, Visibility) {
	zr := image.Rectangle{}
	pb, ok := header1PenBounds(d, offset)
	// not visible
	if !ok {
		return zr, notVisible
	}
	pr := pb.ToRectFloorCeil()

	// allow intersection of empty x in penbounds (case of eof)
	if pr.Dx() == 0 {
		pr.Max.X = pr.Min.X + 1
	}

	ir := d.bounds.Intersect(pr)
	if ir.Empty() {
		if pr.Min.Y < d.bounds.Min.Y {
			return pr, topNotVisible
		} else {
			return pr, bottomNotVisible
		}
	}
	// partially visible
	if ir != pr {
		if pr.Min.Y < d.bounds.Min.Y {
			return pr, topPartVisible
		} else {
			return pr, bottomPartVisible
		}
	}
	// fully visible
	return pr, fullyVisible
}
