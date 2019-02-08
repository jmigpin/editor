package drawer4

import (
	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/mathutil"
)

type RuneOffset struct {
	d *Drawer
}

func (ro *RuneOffset) Init() { ro.init2() }
func (ro *RuneOffset) Iter() { _ = ro.d.iterNext() }
func (ro *RuneOffset) End()  {}

//----------

func (ro *RuneOffset) init2() {
	if ro.d.Opt.RuneOffset.On {
		ls, _ := ro.lineIndexes()
		ro.d.st.runeR.ri = ls

		py := ro.offsetLinePercentPenY()
		ro.d.st.runeR.pen.Y -= py
	}
}

//----------

func (ro *RuneOffset) lineIndexes() (int, int) {
	opt := &ro.d.Opt.RuneOffset
	recalc := opt.line2.calc.offset != opt.offset ||
		opt.line2.calc.measureId != ro.d.measureId
	if recalc {
		opt.line2.calc.offset = opt.offset
		opt.line2.calc.measureId = ro.d.measureId
		lsi, err := iout.LineStartIndex(ro.d.reader, opt.offset)
		if err != nil {
			lsi = 0
		}
		lei, nl, err := iout.LineEndIndex(ro.d.reader, opt.offset)
		if err != nil {
			lei = 0
		}
		_ = nl
		if nl {
			lei--
		}
		opt.line2.start = lsi
		opt.line2.end = lei
	}
	return opt.line2.start, opt.line2.end
}

func (ro *RuneOffset) offsetLinePercentPenY() mathutil.Intf {
	ls, le := ro.lineIndexes()
	o := ro.d.Opt.RuneOffset.offset - ls
	n := le - ls
	if n == 0 {
		// zero until the offset passes the line completely (keeps the line visible)
		return 0
	}
	posPerc := float64(o) / float64(n)
	sy := ro.d.measureContent(ls, n).Y
	return mathutil.Intf(posPerc * float64(sy))
}
