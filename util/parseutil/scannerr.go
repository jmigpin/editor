package parseutil

import "github.com/jmigpin/editor/util/iout/iorw"

// scanner wrapper to be use with iorw.ReaderAt. Needed because the scanner deals with a []byte src, and the position starts at zero. While the reader first position is at r.Min(), where reading a position less then r.Min() gives error.
type ScannerR struct {
	*Scanner
	R iorw.ReaderAt
}

func NewScannerR(r iorw.ReaderAt, index int) *ScannerR {
	sc := &ScannerR{R: r}
	sc.Scanner = NewScanner()

	src, err := iorw.ReadFastFull(sc.R)
	if err != nil {
		return sc // empty scanner, no error (nothing todo, best effort)
	}
	sc.Scanner.SetSrc(src)

	sc.SetPos(index)

	return sc
}
func (sc *ScannerR) SetPos(v int) {
	sc.Scanner.Pos = sc.filterLimit(v - sc.R.Min())
}
func (sc *ScannerR) filterLimit(v int) int {
	if v > sc.R.Max() {
		return sc.R.Max()
	}
	return v
}
func (sc *ScannerR) Pos() int {
	//return sc.filterLimit(sc.R.Min() + sc.Scanner.Pos)
	return sc.R.Min() + sc.Scanner.Pos
}
func (sc *ScannerR) KeepPos() ScannerRPos {
	return ScannerRPos{sc: sc, Pos: sc.Pos()}
}

func (sc *ScannerR) Src(donotuse int)    { panic("!") }
func (sc *ScannerR) SetSrc(donotuse int) { panic("!") }

//----------
//----------
//----------

type ScannerRPos struct {
	sc  *ScannerR
	Pos int
}

func (scp *ScannerRPos) Restore() {
	scp.sc.SetPos(scp.Pos)
}
func (scp *ScannerRPos) IsEmpty() bool {
	return scp.Pos == scp.sc.Pos()
}
func (scp *ScannerRPos) StartEnd() (int, int) {
	start, end := scp.Pos, scp.sc.Pos()
	if start > end { // support reverse mode
		start, end = end, start
	}
	return start, end
}
func (scp *ScannerRPos) Bytes() []byte {
	start, end := scp.StartEnd()
	start -= scp.sc.R.Min()
	end -= scp.sc.R.Min()
	return scp.sc.Scanner.Src[start:end]
}
