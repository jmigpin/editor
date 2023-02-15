package pscan // position scanner

import (
	"errors"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/jmigpin/editor/util/mathutil"
)

//go:generate go run ./wrapgen/wrapgen.go

// NOTE: to debug, use fatal error wrappers
type Scanner struct {
	src       []byte
	srcOffset int // src[0] position
	Reverse   bool
	M         *Match
	W         *Wrap
}

func NewScanner() *Scanner {
	sc := &Scanner{}
	sc.M = &Match{}
	sc.W = &Wrap{}
	sc.M.init(sc)
	sc.W.init(sc)
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
	return mathutil.Max(sc.SrcMin(), mathutil.Min(i, sc.SrcLen()))
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
func (sc *Scanner) SrcLen() int {
	return sc.srcOffset + len(sc.src)
}

//----------

// WARNING: use with caution, using pos in resulting []byte might fail when there is offset
func (sc *Scanner) SrcFullFromOffset() []byte {
	return sc.SrcFromTo(sc.SrcMin(), sc.SrcLen())
}

//----------

func (sc *Scanner) srcSection0(pos int, maxLen int) string {
	start := mathutil.Max(pos-maxLen, sc.SrcMin())
	end := mathutil.Min(pos+maxLen, sc.SrcLen())
	src := sc.SrcFromTo(start, end)
	return SurroundingString(src, pos-start, maxLen)
}
func (sc *Scanner) SrcSection(pos int) string {
	return sc.srcSection0(pos, 35)
}
func (sc *Scanner) SrcError(pos int, err error) error {
	return fmt.Errorf("%v: %v", err, sc.SrcSection(pos))
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
		if pos >= sc.SrcLen() {
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

func (sc *Scanner) EnsureFatalError(err error) error {
	e2, ok := err.(*Error)
	if !ok {
		e2 = &Error{err: err}
	}
	e2.Fatal = true
	return e2
}

//----------

func (sc *Scanner) NewValueKeeper() *ValueKeeper {
	return sc.NewValueKeepers(1)[0]
}
func (sc *Scanner) NewValueKeepers(n int) []*ValueKeeper {
	w := make([]*ValueKeeper, n, n)
	for i := 0; i < n; i++ {
		w[i] = &ValueKeeper{sc: sc}
	}
	return w
}

//----------
//----------
//----------

type ValueKeeper struct {
	sc *Scanner
	V  any // value
}

func (vk *ValueKeeper) KeepValue(pos int, fn VFn) (int, error) {
	vk.V = nil
	if v, p2, err := fn(pos); err != nil {
		return p2, err
	} else {
		vk.V = v
		return p2, nil
	}
}
func (vk *ValueKeeper) WKeepValue(fn VFn) MFn {
	return func(pos int) (int, error) {
		return vk.KeepValue(pos, fn)
	}
}

//----------
//----------
//----------

type MFn func(pos int) (int, error)      // match func
type VFn func(pos int) (any, int, error) // value func

//----------
//----------
//----------

type Error struct {
	err   error
	Fatal bool
}

func (e Error) Error() string {
	return e.err.Error()
}

//----------

func errorIsFatal(err error) bool {
	e, ok := err.(*Error)
	return ok && e.Fatal
}

//----------
//----------
//----------

var NoMatchErr = errors.New("no match")
var EOF = io.EOF
var SOF = errors.New("SOF") // start-of-file (as opposed to EOF)

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
