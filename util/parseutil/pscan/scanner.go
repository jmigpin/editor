package pscan // position scanner

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"unicode/utf8"
)

//go:generate go run ./wrapgen/wrapgen.go

// NOTE: to debug, use match.print* and fatal error wrappers
// NOTE: to debug, activate DebugLimitedLoop

type Scanner struct {
	src       []byte
	srcOffset int // src[0] position
	Reverse   bool
	Debug     bool

	M *Match
	W *Wrap
}

func NewScanner() *Scanner {
	sc := &Scanner{}
	sc.M = &Match{}
	sc.W = &Wrap{}
	sc.M.init(sc)
	sc.W.init(sc)

	//sc.DebugLoop = true

	return sc
}

//----------

func (sc *Scanner) SetSrc(src []byte) {
	sc.src = src
}
func (sc *Scanner) SetSrc2(src []byte, offset int) {
	sc.SetSrc(src)
	sc.srcOffset = offset
}

//----------

func (sc *Scanner) ValidPos(i int) int {
	return max(sc.SrcMin(), min(i, sc.SrcMax()))
}

//----------

func (sc *Scanner) srcAt(i int) int {
	return i - sc.srcOffset
}
func (sc *Scanner) SrcByte(i int) byte {
	return sc.src[sc.srcAt(i)]
}
func (sc *Scanner) SrcFrom(a int) []byte {
	return sc.src[sc.srcAt(a):]
}
func (sc *Scanner) SrcTo(b int) []byte {
	return sc.src[:sc.srcAt(b)]
}
func (sc *Scanner) SrcFromTo(a, b int) []byte {
	if a > b { // support reverse mode
		a, b = b, a
	}
	return sc.src[sc.srcAt(a):sc.srcAt(b)]
}
func (sc *Scanner) SrcMin() int {
	return sc.srcOffset
}
func (sc *Scanner) SrcMax() int {
	return sc.srcOffset + len(sc.src)
}

//----------

// WARNING: use with caution, using pos in resulting []byte might fail when there is offset
// func (sc *Scanner) SrcFullFromOffset() []byte {
// return sc.SrcFromTo(sc.SrcMin(), sc.SrcMax())
func (sc *Scanner) RawSrc() []byte {
	return sc.src
}

//----------

func (sc *Scanner) SrcSection(pos int) string {
	maxLen := 35
	return SurroundingString(sc.src, pos-sc.SrcMin(), maxLen)
}
func (sc *Scanner) SrcError(pos int, err error) error {
	lc := ""
	if l, c, ok := sc.SrcLineCol(pos); ok {
		lc = fmt.Sprintf("%v:%v: ", l, c)
	}
	return fmt.Errorf("%v%v: %q", lc, err, sc.SrcSection(pos))
}
func (sc *Scanner) SrcLineCol(pos int) (int, int, bool) {
	return FindLineColumn(sc.src, pos-sc.SrcMin())
}

//----------

func (sc *Scanner) ReadByte(pos int) (byte, int, error) {
	if sc.Reverse {
		if pos <= 0 {
			return 0, pos, SOF
		}
		pos -= 1
		b := sc.SrcByte(pos)
		return b, pos, nil
	} else {
		if pos >= sc.SrcMax() {
			return 0, pos, EOF
		}
		b := sc.SrcByte(pos)
		pos += 1
		return b, pos, nil
	}
}

func (sc *Scanner) ReadRune(pos int) (rune, int, error) {
	if sc.Reverse {
		ru, size := utf8.DecodeLastRune(sc.SrcTo(pos))
		if size == 0 {
			return 0, pos, SOF
		}
		pos -= size
		return ru, pos, nil
	} else {
		ru, size := utf8.DecodeRune(sc.SrcFrom(pos))
		if size == 0 {
			return 0, pos, EOF
		}
		pos += size
		return ru, pos, nil
	}
}

//----------
//----------
//----------

type MFn = func(pos int) (int, error)      // match func
type VFn = func(pos int) (any, int, error) // value func

//----------
//----------
//----------

func Keep[T any](pos int, v *T, fn VFn) (int, error) {
	p2, err := keep2[T](pos, v, fn)
	if err != nil {
		return p2, err
	}

	// set position
	if u, ok := any(v).(interface{ SetPos(pos int) }); ok {
		u.SetPos(pos)
	}
	if u, ok := any(v).(interface{ SetPosEnd(pos, end int) }); ok {
		u.SetPosEnd(pos, p2)
	}

	return p2, nil
}
func keep2[T any](pos int, v *T, fn VFn) (int, error) {
	v2, p2, err := fn(pos)
	if err != nil {
		return p2, err
	}

	if u, ok := any(v).(interface{ Keep(any) error }); ok {
		if err := u.Keep(v2); err != nil {
			return p2, err
		}
		return p2, nil
	}

	v3, ok := v2.(T)
	if ok {
		*v = v3
		return p2, nil
	}

	// special case: ex: assign string to *string
	rv2 := reflect.ValueOf(v2)
	v2ptr := reflect.New(rv2.Type())
	v2ptr.Elem().Set(rv2)
	v5, ok := (v2ptr.Interface()).(T)
	if ok {
		*v = v5
		return p2, nil
	}

	var zero T
	err = fmt.Errorf("pscan.keep: type is %T, not %T", v2, zero)
	return 0, FatalError(err)
}
func WKeep[T any](v *T, fn VFn) MFn {
	return func(pos int) (int, error) {
		return Keep(pos, v, fn)
	}
}

//----------
//----------
//----------

type Error struct {
	err   error
	Fatal bool
}

func FatalError(err error) error {
	//e2, ok := err.(*Error)
	//if !ok {
	//	e2 = &Error{err: err}
	//}
	//e2.Fatal = true
	//return e2

	if IsFatalError(err) {
		return err
	}
	return &Error{err, true}
}

func (e *Error) Unwrap() error {
	return e.err
}
func (e *Error) Error() string {
	return e.err.Error()
}

//----------

func IsFatalError(err error) bool {
	//e, ok := err.(*Error)
	//return ok && e.Fatal

	e := &Error{}
	if errors.As(err, &e) {
		return e.Fatal
	}
	return false
}

//----------
//----------
//----------

type RuneRange [2]rune // assume [0]<[1]

func (rr RuneRange) HasRune(ru rune) bool {
	return ru >= rr[0] && ru <= rr[1]
}
func (rr RuneRange) IntersectsRange(rr2 RuneRange) bool {
	noIntersection := rr2[1] <= rr[0] || rr2[0] > rr[1]
	return !noIntersection
}
func (rr RuneRange) String() string {
	return fmt.Sprintf("%q-%q", rr[0], rr[1])
}

//----------
//----------
//----------

type AndOpt struct {
	Reverse   *bool
	OptSpaces *SpacesOpt // optional spaces, nil=no optional spaces
}

//----------

type SpacesOpt struct {
	IncludeNL bool
	Esc       rune
}                                     //
func (opt SpacesOpt) HasEscape() bool { return opt.Esc != 0 }

//----------
//----------
//----------

var NoMatchErr = errors.New("no match")
var EOF = io.EOF
var SOF = errors.New("SOF") // start-of-file (as opposed to EOF)
