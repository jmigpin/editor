package statemach

import (
	"fmt"

	"github.com/jmigpin/editor/util/iout/iorw"
)

const eos = -1
const readErr = -2

type Scanner struct {
	Start int
	Pos   int
	R     iorw.Reader
	Match Matcher

	Reverse bool // read direction
}

func NewScanner(r iorw.Reader) *Scanner {
	sc := &Scanner{R: r}
	sc.Match = Matcher{sc: sc}
	sc.SetStartPos(r.Min())
	return sc
}

//----------

func (sc *Scanner) Advance() {
	sc.Start = sc.Pos
}

func (sc *Scanner) SetStartPos(v int) {
	sc.Start = v
	sc.Pos = v
}

func (sc *Scanner) Empty() bool {
	return sc.Start == sc.Pos
}

//----------

func (sc *Scanner) ReadRune() rune {
	if sc.Reverse {
		if sc.Pos <= sc.R.Min() {
			return eos
		}
		ru, w, err := sc.R.ReadLastRuneAt(sc.Pos)
		if err != nil {
			return readErr
		}
		sc.Pos -= w
		return ru
	}

	if sc.Pos >= sc.R.Max() {
		return eos
	}
	ru, w, err := sc.R.ReadRuneAt(sc.Pos)
	if err != nil {
		return readErr
	}
	sc.Pos += w
	return ru
}

//----------

func (sc *Scanner) PeekRune() rune {
	p := sc.Pos
	ru := sc.ReadRune()
	sc.Pos = p
	return ru
}

//----------

func (sc *Scanner) RewindOnFalse(fn func() bool) bool {
	pos := sc.Pos
	if fn() {
		return true
	}
	sc.Pos = pos
	return false
}

//----------

func (sc *Scanner) Value() string {
	start, pos := sc.Start, sc.Pos
	if sc.Reverse {
		start, pos = pos, start
	}
	b, err := sc.R.ReadNSliceAt(start, pos-start)
	if err != nil {
		return ""
	}
	return string(b)
}

//----------

func (sc *Scanner) Errorf(f string, args ...interface{}) error {
	// just n in each direction for error string
	pad := 15
	a := sc.Pos - pad
	if a < sc.R.Min() {
		a = sc.R.Min()
	}
	b := sc.Pos + pad
	if b > sc.R.Max() {
		b = sc.R.Max()
	}

	// context string
	v, err := sc.R.ReadNSliceAt(a, b-a)
	if err != nil {
		return err
	}
	ctx := string(v)
	// position indicator
	p := sc.Pos - a
	ctx = ctx[:p] + "¶" + ctx[p:]
	if a > sc.R.Min() {
		ctx = "..." + ctx
	}
	if b < sc.R.Max() {
		ctx = ctx + "..."
	}

	msg := fmt.Sprintf(f, args...)
	return fmt.Errorf("%s: pos=%v %q", msg, sc.Pos, ctx)
}
