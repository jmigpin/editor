package parseutil

import (
	"fmt"
	"io"
	"regexp"
	"unicode"

	"github.com/jmigpin/editor/util/iout"
)

// scanner matcher utility funcs
type SMatcher struct {
	sc    *Scanner
	cache struct {
		regexps map[string]*regexp.Regexp
	}
}

func (m *SMatcher) init(sc *Scanner) {
	m.sc = sc
	m.cache.regexps = map[string]*regexp.Regexp{}
}

//----------

func (m *SMatcher) Eof() bool {
	pos0 := m.sc.KeepPos()
	_, err := m.sc.ReadRune()
	if err == nil {
		pos0.Restore()
		return false
	}
	return err == io.EOF
}

//----------

func (m *SMatcher) Rune(ru rune) error {
	return m.sc.RestorePosOnErr(func() error {
		ru2, err := m.sc.ReadRune()
		if err != nil {
			return err
		}
		if ru2 != ru {
			return NoMatchErr
		}
		return nil
	})
}
func (m *SMatcher) RuneAny(rs []rune) error { // "or", any of the runes
	return m.sc.RestorePosOnErr(func() error {
		ru, err := m.sc.ReadRune()
		if err != nil {
			return err
		}
		if !ContainsRune(rs, ru) {
			return NoMatchErr
		}
		return nil
	})
}
func (m *SMatcher) RuneAnyNot(rs []rune) error { // "or", any of the runes
	return m.sc.RestorePosOnErr(func() error {
		ru, err := m.sc.ReadRune()
		if err != nil {
			return err
		}
		if ContainsRune(rs, ru) {
			return NoMatchErr
		}
		return nil
	})
}
func (m *SMatcher) RuneSequence(seq []rune) error {
	return m.sc.RestorePosOnErr(func() error {
		for i, l := 0, len(seq); i < l; i++ {
			ru := seq[i]
			if m.sc.Reverse {
				ru = seq[l-1-i]
			}

			// NOTE: using spm.Rune() would call keeppos n times

			ru2, err := m.sc.ReadRune()
			if err != nil {
				return err
			}
			if ru2 != ru {
				return NoMatchErr
			}
		}
		return nil
	})
}
func (m *SMatcher) RuneSequenceMid(rs []rune) error {
	return m.sc.RestorePosOnErr(func() error {
		for k := 0; ; k++ {
			if err := m.RuneSequence(rs); err == nil {
				return nil // match
			}
			if k+1 >= len(rs) {
				break
			}
			// backup to previous rune to try to match again
			m.sc.Reverse = !m.sc.Reverse
			_, err := m.sc.ReadRune()
			m.sc.Reverse = !m.sc.Reverse
			if err != nil {
				return err
			}
		}
		return NoMatchErr
	})
}
func (m *SMatcher) RuneRange(rr RuneRange) error {
	return m.sc.RestorePosOnErr(func() error {
		ru, err := m.sc.ReadRune()
		if err != nil {
			return err
		}
		if !rr.HasRune(ru) {
			return NoMatchErr
		}
		return nil
	})
}
func (m *SMatcher) RuneRangeNot(rr RuneRange) error { // negation
	return m.sc.RestorePosOnErr(func() error {
		ru, err := m.sc.ReadRune()
		if err != nil {
			return err
		}
		if rr.HasRune(ru) {
			return NoMatchErr
		}
		return nil
	})
}
func (m *SMatcher) RunesAndRuneRanges(rs []rune, rrs RuneRanges) error { // negation
	return m.sc.RestorePosOnErr(func() error {
		ru, err := m.sc.ReadRune()
		if err != nil {
			return err
		}
		if !ContainsRune(rs, ru) && !rrs.HasRune(ru) {
			return NoMatchErr
		}
		return nil
	})
}
func (m *SMatcher) RunesAndRuneRangesNot(rs []rune, rrs RuneRanges) error {
	return m.sc.RestorePosOnErr(func() error {
		ru, err := m.sc.ReadRune()
		if err != nil {
			return err
		}
		if ContainsRune(rs, ru) || rrs.HasRune(ru) {
			return NoMatchErr
		}
		return nil
	})
}

//----------

func (m *SMatcher) RuneFn(fn func(rune) bool) error {
	pos0 := m.sc.KeepPos()
	ru, err := m.sc.ReadRune()
	if err == nil {
		if !fn(ru) {
			pos0.Restore()
			err = NoMatchErr
		}
	}
	return err
}

// one or more
func (m *SMatcher) RuneFnLoop(fn func(rune) bool) error {
	for first := true; ; first = false {
		if err := m.RuneFn(fn); err != nil {
			if first {
				return err
			}
			return nil
		}
	}
}

//func (m *SMatcher) RuneFnZeroOrMore(fn func(rune) bool) int {
//	for i := 0; ; i++ {
//		if err := m.RuneFn(fn); err != nil {
//			return i
//		}
//	}
//}
//func (m *SMatcher) RuneFnOneOrMore(fn func(rune) bool) error {
//	return m.LoopRuneFn(fn)

//	if err := m.RuneFn(fn); err != nil {
//		return err
//	}
//	_ = m.RuneFnZeroOrMore(fn)
//	return nil
//}

//----------

// same as rune sequence, but directly using strings comparison
func (m *SMatcher) Sequence(seq string) error {
	if m.sc.Reverse {
		return m.RuneSequence([]rune(seq))
	}
	l := len(seq)
	b := m.sc.Src[m.sc.Pos:]
	if l > len(b) {
		return NoMatchErr
	}
	if string(b[:l]) != seq {
		return NoMatchErr
	}
	m.sc.Pos += l
	return nil
}

//----------

func (m *SMatcher) RegexpFromStartCached(res string) error {
	return m.RegexpFromStart(res, true, 1000)
}
func (m *SMatcher) RegexpFromStart(res string, cache bool, maxLen int) error {
	// TODO: reverse

	res = "^(" + res + ")" // from start

	re := (*regexp.Regexp)(nil)
	if cache {
		re2, ok := m.cache.regexps[res]
		if ok {
			re = re2
		}
	}
	if re == nil {
		re3, err := regexp.Compile(res)
		if err != nil {
			return err
		}
		re = re3
		if cache {
			m.cache.regexps[res] = re
		}
	}

	// limit input to be read
	src := m.sc.Src[m.sc.Pos:]
	max := maxLen
	if max > len(src) {
		max = len(src)
	}
	src = m.sc.Src[m.sc.Pos : m.sc.Pos+max]

	locs := re.FindIndex(src)
	if len(locs) == 0 {
		return NoMatchErr
	}
	m.sc.Pos += locs[1]
	return nil
}

//----------

func (m *SMatcher) DoubleQuotedString(maxLen int) error {
	return m.StringSection("\"", '\\', true, maxLen, false)
}
func (m *SMatcher) QuotedString() error {
	//return m.QuotedString2('\\', 3000, 10)
	return m.QuotedString2('\\', 3000, 3000)
}

// allows escaped runes (if esc!=0)
func (m *SMatcher) QuotedString2(esc rune, maxLen1, maxLen2 int) error {
	// doublequote: fail on newline, eof doesn't close
	if err := m.StringSection("\"", esc, true, maxLen1, false); err == nil {
		return nil
	}
	// singlequote: fail on newline, eof doesn't close (usually a smaller maxlen)
	if err := m.StringSection("'", esc, true, maxLen2, false); err == nil {
		return nil
	}
	// backquote: can have newline, eof doesn't close
	if err := m.StringSection("`", esc, false, maxLen1, false); err == nil {
		return nil
	}
	return fmt.Errorf("not a quoted string")
}

func (m *SMatcher) StringSection(openclose string, esc rune, failOnNewline bool, maxLen int, eofClose bool) error {
	return m.Section(openclose, openclose, esc, failOnNewline, maxLen, eofClose)
}

// match opened/closed sections.
func (m *SMatcher) Section(open, close string, esc rune, failOnNewline bool, maxLen int, eofClose bool) error {
	pos0 := m.sc.Pos
	return m.sc.RestorePosOnErr(func() error {
		if err := m.Sequence(open); err != nil {
			return err
		}
		for {
			if esc != 0 && m.EscapeAny(esc) == nil {
				continue
			}
			if err := m.Sequence(close); err == nil {
				return nil // ok
			}
			// consume rune
			ru, err := m.sc.ReadRune()
			if err != nil {
				// extension: stop on eof
				if eofClose && err == io.EOF {
					return nil // ok
				}

				return err
			}
			// extension: stop after maxlength
			if maxLen > 0 {
				d := m.sc.Pos - pos0
				if d < 0 { // handle reverse
					d = -d
				}
				if d > maxLen {
					return fmt.Errorf("passed maxlen")
				}
			}
			// extension: newline
			if failOnNewline && ru == '\n' {
				return fmt.Errorf("found newline")
			}
		}
	})
}

//----------

func (m *SMatcher) EscapeAny(escape rune) error {
	return m.sc.RestorePosOnErr(func() error {
		if m.sc.Reverse {
			if err := m.NRunes(1); err != nil {
				return err
			}
			return m.Rune(escape)
		}
		if err := m.Rune(escape); err != nil {
			return err
		}
		return m.NRunes(1)
	})
}
func (m *SMatcher) NRunes(n int) error {
	pos0 := m.sc.KeepPos()
	for i := 0; i < n; i++ {
		_, err := m.sc.ReadRune()
		if err != nil {
			pos0.Restore()
			return err
		}
	}
	return nil
}

//----------

func (m *SMatcher) SpacesIncludingNL() bool {
	err := m.Spaces(true, 0)
	return err == nil
}
func (m *SMatcher) SpacesExcludingNL() bool {
	err := m.Spaces(false, 0)
	return err == nil
}
func (m *SMatcher) Spaces(includeNL bool, escape rune) error {
	for first := true; ; first = false {
		if escape != 0 {
			if err := m.EscapeAny(escape); err == nil {
				continue
			}
		}
		pos0 := m.sc.KeepPos()
		ru, err := m.sc.ReadRune()
		if err == nil {
			valid := unicode.IsSpace(ru) && (includeNL || ru != '\n')
			if !valid {
				err = NoMatchErr
			}
		}
		if err != nil {
			pos0.Restore()
			if first {
				return err
			}
			return nil
		}
	}
}

//----------

func (m *SMatcher) FnOptional(fn func() error) error {
	pos0 := m.sc.KeepPos()
	if err := fn(); err != nil {
		pos0.Restore()
	}
	return nil
}
func (m *SMatcher) FnOr(fns ...func() error) error {
	me := iout.MultiError{}
	for _, fn := range fns {
		if err := fn(); err != nil {
			me.Add(err)
			continue
		}
		return nil
	}
	return me.Result()
}
func (m *SMatcher) FnAnd(fns ...func() error) error {
	if m.sc.Reverse {
		for i := len(fns) - 1; i >= 0; i-- {
			fn := fns[i]
			if err := fn(); err != nil {
				return err
			}
		}
	} else {
		for _, fn := range fns {
			if err := fn(); err != nil {
				return err
			}
		}
	}
	return nil
}

//----------

func (m *SMatcher) ToNLExcludeOrEnd(esc rune) int {
	pos0 := m.sc.KeepPos()
	valid := func(ru rune) bool { return ru != '\n' }
	for {
		if esc != 0 && m.EscapeAny(esc) == nil {
			continue
		}
		if err := m.RuneFn(valid); err == nil {
			continue
		}
		break
	}
	return pos0.Len()
}
func (m *SMatcher) ToNLIncludeOrEnd(esc rune) int {
	pos0 := m.sc.KeepPos()
	_ = m.ToNLExcludeOrEnd(esc)
	_ = m.Rune('\n')
	return pos0.Len()
}

//----------

func (m *SMatcher) Digit() error {
	return m.RuneFn(unicode.IsDigit)
}
func (m *SMatcher) Digits() error {
	return m.RuneFnLoop(unicode.IsDigit)
}
func (m *SMatcher) Integer() error {
	// TODO: reverse

	//u := "[+-]?[0-9]+"
	//return m.RegexpFromStartCached(u)

	return m.sc.RestorePosOnErr(func() error {
		return m.FnAnd(
			func() error {
				_ = m.RuneAny([]rune("+-")) // optional
				return nil
			},
			m.Digits,
		)
	})
}
func (m *SMatcher) Float() error {
	// TODO: reverse
	u := "[+-]?([0-9]*[.])?[0-9]+"
	return m.RegexpFromStartCached(u)
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

type RuneRanges []RuneRange

func (rrs RuneRanges) HasRune(ru rune) bool {
	for _, rr := range rrs {
		if rr.HasRune(ru) {
			return true
		}
	}
	return false
}
