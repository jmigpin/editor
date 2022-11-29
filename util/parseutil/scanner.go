package parseutil

import (
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

type Scanner struct {
	Src []byte
	Pos int
	M   SMatcher
	P   SParser

	Reverse     bool // read direction
	Debug       bool
	ErrFilename string // used for errors only
}

func NewScanner() *Scanner {
	sc := &Scanner{}
	sc.M.init(sc)
	sc.P.init(sc)
	return sc
}

//----------

func (sc *Scanner) SetSrc(src []byte) {
	sc.Src = src
	sc.Pos = 0
}

//----------

func (sc *Scanner) ReadRune() (rune, error) {
	ru := rune(0)
	size := 0
	if sc.Reverse {
		ru, size = utf8.DecodeLastRune(sc.Src[:sc.Pos])
		size = -size // decrease pos
	} else {
		ru, size = utf8.DecodeRune(sc.Src[sc.Pos:])
	}
	if size == 0 {
		return 0, io.EOF
	}
	sc.Pos += size

	if sc.Debug {
		fmt.Printf("%v: %q\n", sc.Pos, ru)
	}

	return ru, nil
}
func (sc *Scanner) PeekRune() (rune, error) {
	pos0 := sc.KeepPos()
	ru, err := sc.ReadRune()
	pos0.Restore()
	return ru, err
}

//----------

func (sc *Scanner) KeepPos() ScannerPos {
	return ScannerPos{sc: sc, Pos: sc.Pos}
}

// from provided, to current position
func (sc *Scanner) BytesFrom(from int) []byte {
	return sc.Src[from:sc.Pos]
}

func (sc *Scanner) RestorePosOnErr(fn func() error) error {
	pos0 := sc.KeepPos()
	if err := fn(); err != nil {
		pos0.Restore()
		return err
	}
	return nil
}

//----------

func (sc *Scanner) SrcErrorf(f string, args ...any) error {
	return sc.SrcError(fmt.Errorf(f, args...))
}
func (sc *Scanner) SrcError(err error) error {
	return sc.SrcError2(err, 20)
}
func (sc *Scanner) SrcError2(err error, maxLen int) error {
	filename := sc.ErrFilename
	if filename == "" {
		filename = "<bytes>"
	}

	// position
	pos := sc.Pos
	if pe, ok := err.(*PosError); ok {
		pos = pe.Pos
	}
	if fe, ok := err.(*SFatalError); ok {
		pos = fe.Pos
	}

	line, col := IndexLineColumn2(sc.Src, pos)
	str := SurroundingString(sc.Src, pos, maxLen)
	return fmt.Errorf("%v:%d:%d: %v: %q", filename, line, col, err, str)
}

//----------
//----------
//----------

type ScannerPos struct {
	sc  *Scanner
	Pos int
}

func (sp *ScannerPos) Restore() {
	sp.sc.Pos = sp.Pos
}
func (sp *ScannerPos) IsEmpty() bool {
	return sp.Pos == sp.sc.Pos
}
func (sp *ScannerPos) Len() int {
	start, end := sp.StartEnd()
	return end - start
}
func (sp *ScannerPos) StartEnd() (int, int) {
	start, end := sp.Pos, sp.sc.Pos
	if start > end { // support reverse mode
		start, end = end, start
	}
	return start, end
}
func (sp *ScannerPos) Bytes() []byte {
	start, end := sp.StartEnd()
	return sp.sc.Src[start:end]
}

//----------
//----------
//----------

type SFatalError struct {
	PosError
}

func IsFatalErr(err error) bool {
	_, ok := err.(*SFatalError)
	return ok
}

//----------

// error with position
type PosError struct {
	error
	Pos int
}

//----------
//----------
//----------

var NoMatchErr = errors.New("no match")
