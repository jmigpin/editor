package scanutil

import (
	"errors"
	"strconv"
	"strings"
	"unicode"
)

type Matcher struct {
	sc *Scanner
}

func (m *Matcher) Rune(ru rune) bool {
	return m.sc.RewindOnFalse(func() bool {
		return m.sc.ReadRune() == ru
	})
}

func (m *Matcher) End() bool {
	return m.Rune(Eof) || m.Rune(readErr)
}

func (m *Matcher) Any(valid string) bool {
	return m.sc.RewindOnFalse(func() bool {
		return strings.ContainsRune(valid, m.sc.ReadRune())
	})
}

func (m *Matcher) Except(invalid string) bool {
	return m.sc.RewindOnFalse(func() bool {
		if m.End() {
			return false
		}
		return !strings.ContainsRune(invalid, m.sc.ReadRune())
	})
}

func (m *Matcher) Sequence(s string) bool {
	if s == "" {
		return false
	}
	if m.sc.Reverse {
		s = ReverseString(s)
	}
	return m.sc.RewindOnFalse(func() bool {
		for _, ru := range s {
			if m.sc.ReadRune() != ru {
				return false
			}
		}
		return true
	})
}

func (m *Matcher) Fn(fn func(rune) bool) bool {
	return m.sc.RewindOnFalse(func() bool {
		if m.End() {
			return false
		}
		return fn(m.sc.ReadRune())
	})
}

// Returns true if at least one rune was read.
func (m *Matcher) FnLoop(fn func(rune) bool) bool {
	v := false
	for {
		if m.Fn(fn) {
			v = true
			continue
		}
		break
	}
	return v
}

// Must all return true.
func (m *Matcher) FnOrder(fns ...func() bool) bool {
	index := func(i int) int {
		if m.sc.Reverse {
			return len(fns) - 1 - i
		}
		return i
	}
	return m.sc.RewindOnFalse(func() bool {
		for i := 0; i < len(fns); i++ {
			fn := fns[index(i)]
			if !fn() {
				return false
			}
		}
		return true
	})
}

//----------

func (m *Matcher) NRunes(n int) bool {
	return m.sc.RewindOnFalse(func() bool {
		c := 0
		_ = m.FnLoop(func(ru rune) bool {
			if c >= n {
				return false // stop loop
			}
			c++
			return true
		})
		return c == n // result
	})
}

func (m *Matcher) NPos(n int) bool {
	if m.sc.Reverse {
		if m.sc.Pos-n < m.sc.R.Min() {
			return false
		}
		m.sc.Pos -= n
		return true
	}

	if m.sc.Pos+n > m.sc.R.Max() {
		return false
	}
	m.sc.Pos += n
	return true
}

//----------

func (m *Matcher) Spaces() bool {
	return m.FnLoop(unicode.IsSpace)
}

func (m *Matcher) SpacesExceptNewline() bool {
	return m.FnLoop(func(ru rune) bool {
		if ru == '\n' {
			return false
		}
		return unicode.IsSpace(ru)
	})
}

func (m *Matcher) ExceptUnescapedSpaces(escape rune) bool {
	pos := m.sc.Pos
	notSpace := func(ru rune) bool { return !unicode.IsSpace(ru) }
	for {
		if m.End() {
			break
		}
		if m.Escape(escape) {
			continue
		}
		if m.Fn(notSpace) {
			continue
		}
		break
	}
	return m.sc.Pos != pos
}

func (m *Matcher) ToNewlineOrEnd() {
	_ = m.FnLoop(func(ru rune) bool {
		return ru != '\n'
	})
}

//----------

func (m *Matcher) Section(open, close string, escape rune, failOnNewline bool, maxLen int, eofClose bool) bool {
	return m.sc.RewindOnFalse(func() bool {
		start := m.sc.Pos

		if !m.Sequence(open) {
			return false
		}
		for {
			if escape != 0 && m.Escape(escape) {
				continue
			}
			if m.Sequence(close) {
				return true
			}
			ru := m.sc.ReadRune() // consume rune

			// extension: stop on eof
			if ru == Eof {
				return eofClose
			}
			// extension: newline
			if failOnNewline && ru == '\n' {
				return false
			}
			// extension: stop on maxlength
			if maxLen > 0 {
				d := m.sc.Pos - start
				if d < 0 {
					d = -d
				}
				if d >= maxLen {
					return false
				}
			}
		}
	})
}

//----------

func (m *Matcher) GoQuotes(escape rune, maxLen, maxLenSingleQuote int) bool {
	if m.Quote('"', escape, true, maxLen) {
		return true
	}
	if m.Quote('`', escape, false, maxLen) {
		return true
	}
	if m.Quote('\'', escape, true, maxLenSingleQuote) {
		return true
	}
	return false
}

func (m *Matcher) Quote(quote rune, escape rune, failOnNewline bool, maxLen int) bool {
	q := string(quote)
	return m.Section(q, q, escape, failOnNewline, maxLen, false)
}

func (m *Matcher) Quoted(validQuotes string, escape rune, failOnNewline bool, maxLen int) bool {
	ru := m.sc.PeekRune()
	if strings.ContainsRune(validQuotes, ru) {
		if m.Quote(ru, escape, failOnNewline, maxLen) {
			return true
		}
	}
	return false
}

func (m *Matcher) DoubleQuoteStr() bool {
	q := string('"')
	return m.Section(q, q, '\\', true, 0, false)
}
func (m *Matcher) SingleQuoteStr() bool {
	q := string('\'')
	return m.Section(q, q, '\\', true, 0, false)
}
func (m *Matcher) MultiLineComment() bool {
	return m.Section("/*", "*/", 0, false, 0, false)
}
func (m *Matcher) LineComment() bool {
	return m.Section("//", "\n", 0, true, 0, false)
}

//----------

func (m *Matcher) Escape(escape rune) bool {
	if m.sc.Reverse {
		return m.reverseEscape(escape)
	}

	return m.sc.RewindOnFalse(func() bool {
		// needs rune to succeed, will fail on eos
		return m.Rune(escape) && m.NRunes(1)
	})
}

func (m *Matcher) reverseEscape(escape rune) bool {
	return m.sc.RewindOnFalse(func() bool {
		if !m.NRunes(1) {
			return false
		}
		// need to read odd number of escapes to accept
		c := 0
		epos := 0
		for {
			if m.Rune(escape) {
				c++
				if c == 1 {
					epos = m.sc.Pos
				} else if c > 10 { // max escapes to test
					return false
				}
			} else {
				if c%2 == 1 { // odd
					m.sc.Pos = epos // epos was set
					return true
				}
				return false
			}
		}
	})
}

//----------

func (m *Matcher) Id() bool {
	if m.sc.Reverse {
		panic("can't parse in reverse")
	}

	if !(m.Any("_") ||
		m.Fn(unicode.IsLetter)) {
		return false
	}
	for m.Any("_-") ||
		m.FnLoop(unicode.IsLetter) ||
		m.FnLoop(unicode.IsDigit) {
	}
	return true
}

func (m *Matcher) Int() bool {
	return m.FnOrder(
		func() bool {
			_ = m.Any("+-")
			return true
		},
		func() bool {
			return m.FnLoop(unicode.IsDigit)
		})
}

func (m *Matcher) Float() bool {
	if m.sc.Reverse {
		panic("can't parse in reverse") // TODO
	}

	return m.sc.RewindOnFalse(func() bool {
		ok := false
		_ = m.Any("+-")
		if m.FnLoop(unicode.IsDigit) {
			ok = true
		}
		if m.Any(".") {
			if m.FnLoop(unicode.IsDigit) {
				ok = true
			}
		}
		ok3 := m.sc.RewindOnFalse(func() bool {
			ok2 := false
			if m.Any("eE") {
				_ = m.Any("+-")
				if m.FnLoop(unicode.IsDigit) {
					ok2 = true
				}
			}
			return ok2
		})
		return ok || ok3
	})
}

//----------

func (m *Matcher) IntValue() (int, error) {
	if !m.Int() {
		return 0, errors.New("failed to parse int")
	}
	return strconv.Atoi(string(m.sc.Value()))
}

func (m *Matcher) FloatValue() (float64, error) {
	if !m.Float() {
		return 0, errors.New("failed to parse float")
	}
	return strconv.ParseFloat(string(m.sc.Value()), 64)
}

//----------

func (m *Matcher) IntValueAdvance() (int, error) {
	v, err := m.IntValue()
	if err != nil {
		return 0, err
	}
	m.sc.Advance()
	return v, nil
}

func (m *Matcher) FloatValueAdvance() (float64, error) {
	v, err := m.FloatValue()
	if err != nil {
		return 0, err
	}
	m.sc.Advance()
	return v, nil
}

//----------

func (m *Matcher) SpacesAdvance() error {
	if !m.Spaces() {
		return m.sc.Errorf("expecting space")
	}
	m.sc.Advance()
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
