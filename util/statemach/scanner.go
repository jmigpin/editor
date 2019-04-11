package statemach

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/jmigpin/editor/util/iout/iorw"
)

//const eos = -1
//const readErr = -2

type Scanner struct {
	Start int
	Pos   int
	R     iorw.Reader
	Match Matcher

	reverse bool
}

func NewScanner(r iorw.Reader) *Scanner {
	sc := &Scanner{R: r}
	sc.Match = Matcher{sc: sc}

	min := r.Min()
	sc.Start = min
	sc.Pos = min

	return sc
}

//----------

func (sc *Scanner) RevertReadDirection() {
	sc.reverse = !sc.reverse
}

//----------

func (sc *Scanner) ReadRune() rune {
	if sc.reverse {
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

func (sc *Scanner) Advance() {
	sc.Start = sc.Pos
}

func (sc *Scanner) Empty() bool {
	return sc.Start == sc.Pos
}

func (sc *Scanner) RewindOnFalse(fn func() bool) bool {
	pos := sc.Pos
	if fn() {
		return true
	}
	sc.Pos = pos
	return false
}

//----------

func (sc *Scanner) PeekRune() rune {
	p := sc.Pos
	ru := sc.ReadRune()
	sc.Pos = p
	return ru
}

//----------

func (sc *Scanner) Value() string {
	start, pos := sc.Start, sc.Pos
	if sc.reverse {
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
	return fmt.Errorf("%s: pos=%v [%v]", msg, sc.Pos, ctx)
}

//----------
//----------
//----------

type Matcher struct {
	sc *Scanner
}

func (p *Matcher) Rune(ru rune) bool {
	return p.sc.RewindOnFalse(func() bool {
		return p.sc.ReadRune() == ru
	})
}

func (p *Matcher) End() bool {
	return p.Rune(eos) || p.Rune(readErr)
}

func (p *Matcher) Any(valid string) bool {
	return p.sc.RewindOnFalse(func() bool {
		return strings.ContainsRune(valid, p.sc.ReadRune())
	})
}

func (p *Matcher) Except(invalid string) bool {
	return p.sc.RewindOnFalse(func() bool {
		return !strings.ContainsRune(invalid, p.sc.ReadRune())
	})
}

func (p *Matcher) Sequence(s string) bool {
	if s == "" {
		return false
	}
	if p.sc.reverse {
		s = ReverseString(s)
	}
	return p.sc.RewindOnFalse(func() bool {
		for _, ru := range s {
			if p.sc.ReadRune() != ru {
				return false
			}
		}
		return true
	})
}

func (p *Matcher) Fn(fn func(rune) bool) bool {
	return p.sc.RewindOnFalse(func() bool {
		if p.End() {
			return false
		}
		return fn(p.sc.ReadRune())
	})
}

// Returns true if at least one rune was read.
func (p *Matcher) FnLoop(fn func(rune) bool) bool {
	v := false
	for {
		if p.Fn(fn) {
			v = true
			continue
		}
		break
	}
	return v
}

//----------

func (p *Matcher) NRunes(n int) bool {
	return p.sc.RewindOnFalse(func() bool {
		c := 0
		_ = p.FnLoop(func(ru rune) bool {
			if c >= n {
				return false // stop loop
			}
			c++
			return true
		})
		return c == n // result
	})
}

func (p *Matcher) NPos(n int) bool {
	if p.sc.reverse {
		if p.sc.Pos-n < p.sc.R.Min() {
			return false
		}
		p.sc.Pos -= n
		return true
	}

	if p.sc.Pos+n > p.sc.R.Max() {
		return false
	}
	p.sc.Pos += n
	return true
}

//----------

func (p *Matcher) Spaces() bool {
	return p.FnLoop(unicode.IsSpace)
}

func (p *Matcher) SpacesExceptNewline() bool {
	return p.FnLoop(func(ru rune) bool {
		if ru == '\n' {
			return false
		}
		return unicode.IsSpace(ru)
	})
}

func (p *Matcher) ToNewlineOrEnd() {
	_ = p.FnLoop(func(ru rune) bool {
		return ru != '\n'
	})
}

//----------

func (p *Matcher) Quoted(validQuotes string, escape rune) bool {
	ru := p.sc.PeekRune()
	if strings.ContainsRune(validQuotes, ru) {
		if p.Quote(ru, escape) {
			return true
		}
	}
	return false
}

func (p *Matcher) Quote(quote rune, escape rune) bool {
	return p.Quote2(quote, escape, false, -1)
}

func (p *Matcher) Quote2(quote rune, escape rune, breakOnNewline bool, maxLen int) bool {
	return p.sc.RewindOnFalse(func() bool {
		if !p.Rune(quote) {
			return false
		}
		for {
			if p.End() {
				break
			}
			if p.Escape(escape) {
				continue
			}

			ru := p.sc.ReadRune()

			if ru == quote {
				return true
			}

			if breakOnNewline && ru == '\n' {
				break
			}

			if maxLen > 0 {
				d := p.sc.Pos - p.sc.Start
				if d < 0 {
					d = -d
				}
				if d > maxLen {
					break
				}
			}
		}
		return false
	})
}

//----------

func (p *Matcher) Escape(escape rune) bool {
	if p.sc.reverse {
		return p.sc.RewindOnFalse(func() bool {
			if !p.NRunes(1) {
				return false
			}
			// need to read odd number of escapes to accept
			c := 0
			epos := 0
			for {
				if p.Rune(escape) {
					c++
					if c == 1 {
						epos = p.sc.Pos
					} else if c > 10 { // max escapes to test
						return false
					}
				} else {
					if c%2 == 1 { // odd
						p.sc.Pos = epos // epos was set
						return true
					}
					return false
				}
			}
		})
	}

	return p.sc.RewindOnFalse(func() bool {
		// needs rune to succeed, will fail on eos
		return p.Rune(escape) && p.NRunes(1)
	})
}

//----------

func (p *Matcher) Id() bool {
	if p.sc.reverse {
		panic("can't parse in reverse")
	}

	if !(p.Any("_") ||
		p.Fn(unicode.IsLetter)) {
		return false
	}
	for p.Any("_-") ||
		p.FnLoop(unicode.IsLetter) ||
		p.FnLoop(unicode.IsDigit) {
	}
	return true
}

func (p *Matcher) Int() bool {
	if p.sc.reverse {
		panic("can't parse in reverse")
	}

	return p.sc.RewindOnFalse(func() bool {
		_ = p.Any("+-")
		return p.FnLoop(unicode.IsDigit)
	})
}

func (p *Matcher) Float() bool {
	if p.sc.reverse {
		panic("can't parse in reverse")
	}

	return p.sc.RewindOnFalse(func() bool {
		ok := false
		_ = p.Any("+-")
		if p.FnLoop(unicode.IsDigit) {
			ok = true
		}
		if p.Any(".") {
			if p.FnLoop(unicode.IsDigit) {
				ok = true
			}
		}
		ok3 := p.sc.RewindOnFalse(func() bool {
			ok2 := false
			if p.Any("eE") {
				_ = p.Any("+-")
				if p.FnLoop(unicode.IsDigit) {
					ok2 = true
				}
			}
			return ok2
		})
		return ok || ok3
	})
}

//----------

func (p *Matcher) IntValue() (int, error) {
	if !p.Int() {
		return 0, errors.New("failed to parse int")
	}
	return strconv.Atoi(p.sc.Value())
}

func (p *Matcher) FloatValue() (float64, error) {
	if !p.Float() {
		return 0, errors.New("failed to parse float")
	}
	return strconv.ParseFloat(p.sc.Value(), 64)
}

//----------

func (p *Matcher) IntValueAdvance() (int, error) {
	v, err := p.IntValue()
	if err != nil {
		return 0, err
	}
	p.sc.Advance()
	return v, nil
}

//----------

func (p *Matcher) SpacesAdvance() error {
	if !p.Spaces() {
		return p.sc.Errorf("expecting space")
	}
	p.sc.Advance()
	return nil
}

//----------

func ReverseString(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}
