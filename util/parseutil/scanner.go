package parseutil

import (
	"fmt"
	"io"
	"strconv"
	"unicode/utf8"
)

// not safe to parse concurrently (match/parse uses closures)
type Scanner struct {
	Src []byte
	Pos int
	M   ScMatch
	P   ScParse

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
	if sc.Debug {
		fmt.Printf("%v: %q\n", sc.Pos, ru)
	}
	sc.Pos += size

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

func (sc *Scanner) RestorePosOnErr(fn func() error) error {
	pos0 := sc.KeepPos()
	if err := fn(); err != nil {
		pos0.Restore()
		return err
	}
	return nil
}

//----------

func (sc *Scanner) NewValueKeeper() *ScValueKeeper {
	return &ScValueKeeper{sc: sc}
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
	if pe, ok := err.(*ScPosError); ok {
		pos = pe.Pos
	}
	if fe, ok := err.(*ScFatalError); ok {
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

// error with position
type ScPosError struct {
	Err error
	Pos int
}

func (e *ScPosError) Error() string {
	return e.Err.Error()
}

//----------

type ScFatalError struct {
	ScPosError
}

func IsScFatalError(err error) bool {
	_, ok := err.(*ScFatalError)
	return ok
}

//----------
//----------
//----------

type ScValueKeeper struct {
	sc    *Scanner
	Value any
}

func (vk *ScValueKeeper) Reset() {
	vk.Value = nil
}

//----------

func (vk *ScValueKeeper) KeepBytes(fn ScFn) ScFn {
	return func() error {
		pos0 := vk.sc.KeepPos()
		err := fn()
		vk.Value = pos0.Bytes()
		return err
	}
}
func (vk *ScValueKeeper) KeepValue(fn ScValueFn) ScFn {
	return func() error {
		v, err := fn()
		vk.Value = v
		return err
	}
}

//----------

func (vk *ScValueKeeper) BytesOrNil() []byte {
	if b, ok := vk.Value.([]byte); ok {
		return b
	}
	return nil
}

//----------

func (vk *ScValueKeeper) Int() (int, error) {
	b, ok := vk.Value.([]byte)
	if !ok {
		return 0, fmt.Errorf("not []byte")
	}
	v, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		return 0, err
	}
	return int(v), nil
}
func (vk *ScValueKeeper) IntOrZero() int {
	if v, err := vk.Int(); err == nil {
		return v
	}
	return 0
}

//----------

func (vk *ScValueKeeper) StringOptional() string {
	if vk.Value == nil {
		return ""
	}
	return vk.Value.(string)
}
func (vk *ScValueKeeper) String() string {
	return vk.Value.(string)
}

//----------
//----------
//----------

type ScFn func() error
type ScValueFn func() (any, error)
