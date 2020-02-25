package drawer4

import (
	"github.com/jmigpin/editor/util/iout/iorw"
)

type LineStart struct {
	d *Drawer
}

func (ls *LineStart) Init() {
	st := &ls.d.st.lineStart
	// content line start
	ls.d.st.lineStart.q = nil
	ls.d.st.lineStart.uppedLines = 0
	ls.d.st.runeR.ri = ls.lineStartIndex(st.offset, st.nLinesUp)
}

func (ls *LineStart) Iter() {
	st := &ls.d.st.lineStart
	if ls.d.st.line.lineStart || ls.d.st.lineWrap.postLineWrap {
		st.q = append(st.q, ls.d.st.runeR.ri)
	}
	if ls.d.st.runeR.ri >= st.offset {
		// don't stop before postLineWrap
		if !ls.d.st.lineWrap.preLineWrap {
			ls.d.iterStop()
			return
		}
	}
	if !ls.d.iterNext() {
		return
	}
}

func (ls *LineStart) End() {
	st := &ls.d.st.lineStart
	// count lines back
	if len(st.q) == 0 {
		// worst case ri (start ri)
		st.ri = ls.d.st.runeR.startRi
	} else {
		k := st.nLinesUp
		if k >= len(st.q) {
			k = len(st.q) - 1
		}
		st.ri = st.q[len(st.q)-1-k]
		st.uppedLines = k
	}
}

//----------

func (ls *LineStart) lineStartIndex(offset, nLinesUp int) int {
	w := ls.linesStartIndexes(offset, nLinesUp)

	// read error case
	if len(w) == 0 {
		return offset
	}

	if nLinesUp >= len(w) {
		nLinesUp = len(w) - 1
	}
	return w[nLinesUp]
}

func (ls *LineStart) linesStartIndexes(offset, nLinesUp int) []int {
	// reader
	rd := ls.d.st.lineStart.reader
	if rd == nil {
		rd = ls.d.limitedReaderPad(offset)
	}

	// ensure offset is within max bound
	if offset > rd.Max() {
		offset = rd.Max()
	}

	w := []int{}
	for i := 0; i <= nLinesUp; i++ {
		k, err := iorw.LineStartIndex(rd, offset)
		if err != nil {
			break
		}
		w = append(w, k)
		offset = k - 1
	}
	return w
}
