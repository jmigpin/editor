package pscan

import (
	"fmt"
	"math/bits"
	"regexp"
	"strconv"
	"unicode"
)

type Match struct {
	sc    *Scanner
	W     *Wrap
	cache struct {
		regexps map[string]*regexp.Regexp
	}
}

func (m *Match) init(sc *Scanner) {
	m.sc = sc
	m.W = sc.W
	m.cache.regexps = map[string]*regexp.Regexp{}
}

//----------

// runs in order, even when in reverse mode
func (m *Match) And(pos int, fns ...MFn) (int, error) {
	for _, fn := range fns {
		if p2, err := fn(pos); err != nil {
			return p2, err
		} else {
			pos = p2
		}
	}
	return pos, nil
}

// "and" reversible: in reverse mode, runs the last fn first
func (m *Match) AndR(pos int, fns ...MFn) (int, error) {
	if m.sc.Reverse {
		for i := len(fns) - 1; i >= 0; i-- {
			fn := fns[i]
			if p2, err := fn(pos); err != nil {
				return p2, err
			} else {
				pos = p2
			}
		}
		return pos, nil
	}
	return m.And(pos, fns...)
}

// runs in order, even when in reverse mode
func (m *Match) Or(pos int, fns ...MFn) (int, error) {
	err0 := (error)(nil)
	p0 := -1
	for _, fn := range fns {
		if p2, err := fn(pos); err != nil {
			if errorIsFatal(err) {
				return p2, err
			}
			// keep furthest error
			if err0 == nil ||
				(!m.sc.Reverse && p2 > p0) ||
				(m.sc.Reverse && p2 < p0) {
				p0 = p2
				err0 = err
			}
		} else {
			return p2, nil
		}
	}
	return p0, err0
}

func (m *Match) Optional(pos int, fn MFn) (int, error) {
	if p2, err := fn(pos); err != nil {
		if errorIsFatal(err) {
			return p2, err
		}
		return pos, nil
	} else {
		return p2, nil
	}
}

//----------

func (m *Match) Byte(pos int, b byte) (int, error) {
	b2, p2, err := m.sc.ReadByte(pos)
	if err != nil {
		return p2, err
	}
	if b2 != b {
		return pos, NoMatchErr // position before reading
	}
	return p2, nil
}
func (m *Match) ByteFn(pos int, fn func(byte) bool) (int, error) {
	b, p2, err := m.sc.ReadByte(pos)
	if err != nil {
		return p2, err
	}
	if !fn(b) {
		return pos, NoMatchErr // position before reading
	}
	return p2, nil
}

// one or more
func (m *Match) ByteFnLoop(pos int, fn func(byte) bool) (int, error) {
	return m.Loop(pos, func(p2 int) (int, error) {
		return m.ByteFn(p2, fn)
	})
}

func (m *Match) ByteSequence(pos int, seq []byte) (int, error) {
	for i, l := 0, len(seq); i < l; i++ {
		ru := seq[i]
		if m.sc.Reverse {
			ru = seq[l-1-i]
		}
		if p2, err := m.Byte(pos, ru); err != nil {
			return p2, err
		} else {
			pos = p2
		}
	}
	return pos, nil
}

func (m *Match) NBytesFn(pos int, n int, fn func(byte) bool) (int, error) {
	return m.NLoop(pos, n, m.W.ByteFn(fn))
}
func (m *Match) NBytes(pos int, n int) (int, error) {
	accept := func(byte) bool { return true }
	return m.NBytesFn(pos, n, accept)
}

func (m *Match) OneByte(pos int) (int, error) {
	_, p2, err := m.sc.ReadByte(pos)
	return p2, err
}

//----------

func (m *Match) Rune(pos int, ru rune) (int, error) {
	ru2, p2, err := m.sc.ReadRune(pos)
	if err != nil {
		return p2, err
	}
	if ru2 != ru {
		return pos, NoMatchErr // position before reading
	}
	return p2, nil
}

func (m *Match) RuneFn(pos int, fn func(rune) bool) (int, error) {
	ru, p2, err := m.sc.ReadRune(pos)
	if err != nil {
		return p2, err
	}
	if !fn(ru) {
		return pos, NoMatchErr // position before reading
	}
	return p2, nil
}

// one or more
func (m *Match) RuneFnLoop(pos int, fn func(rune) bool) (int, error) {
	return m.Loop(pos, func(p2 int) (int, error) {
		return m.RuneFn(p2, fn)
	})
}

// previously "RuneAny"
func (m *Match) RuneOneOf(pos int, rs []rune) (int, error) {
	return m.RuneFn(pos, func(ru rune) bool {
		return ContainsRune(rs, ru)
	})
}

// previously "RuneAnyNot"
func (m *Match) RuneNoneOf(pos int, rs []rune) (int, error) {
	return m.RuneFn(pos, func(ru rune) bool {
		return !ContainsRune(rs, ru)
	})
}

func (m *Match) RuneSequence(pos int, seq []rune) (int, error) {
	for i, l := 0, len(seq); i < l; i++ {
		ru := seq[i]
		if m.sc.Reverse {
			ru = seq[l-1-i]
		}
		if p2, err := m.Rune(pos, ru); err != nil {
			return p2, err // returning failing position
		} else {
			pos = p2
		}
	}
	return pos, nil
}
func (m *Match) RuneSequenceMid(pos int, rs []rune) (int, error) {
	p0 := pos
	for k := 0; ; k++ {
		if p2, err := m.RuneSequence(pos, rs); err == nil {
			return p2, nil // match
		}
		if k+1 >= len(rs) { // OK in reverse
			return p0, NoMatchErr
		}
		// backup to previous rune to try to match again
		rev0 := m.sc.Reverse
		m.sc.Reverse = !rev0
		p4, err := m.OneRune(pos)
		m.sc.Reverse = rev0 // restore
		if err != nil {
			return p4, err
		} else {
			pos = p4
		}
	}
}
func (m *Match) NRunesFn(pos int, n int, fn func(rune) bool) (int, error) {
	return m.NLoop(pos, n, m.W.RuneFn(fn))
}
func (m *Match) NRunes(pos int, n int) (int, error) {
	accept := func(rune) bool { return true }
	return m.NRunesFn(pos, n, accept)
}

// equivalent to NRunes(1) but faster
func (m *Match) OneRune(pos int) (int, error) {
	_, p2, err := m.sc.ReadRune(pos)
	return p2, err
}

//----------

func (m *Match) Sequence(pos int, seq string) (int, error) {
	//return m.RuneSequence(pos, []rune(seq))
	return m.ByteSequence(pos, []byte(seq))
}
func (m *Match) SequenceMid(pos int, seq string) (int, error) {
	return m.RuneSequenceMid(pos, []rune(seq))
}

//----------

func (m *Match) RuneRanges(pos int, rrs ...RuneRange) (int, error) {
	if ru, p2, err := m.sc.ReadRune(pos); err != nil {
		return p2, err
	} else {
		for _, rr := range rrs {
			if rr.HasRune(ru) {
				return p2, nil
			}
		}
		return pos, NoMatchErr
	}
}

//----------

// max<=-1 means no upper limit
func (m *Match) LimitedLoop(pos int, min, max int, fn MFn) (int, error) {
	for i := 0; ; i++ {
		if max >= 0 && i >= max {
			return pos, fmt.Errorf("loop max: %v", max)
		}
		p2, err := fn(pos)
		if err != nil {
			if errorIsFatal(err) {
				return p2, err
			}
			if i >= min {
				return pos, nil // last good fn() position
			}
			return p2, err
		}
		pos = p2
	}
	return pos, nil
}

// one or more
func (m *Match) Loop(pos int, fn MFn) (int, error) {
	return m.LimitedLoop(pos, 1, -1, fn)
}

// optional loop: zero or more
func (m *Match) OptLoop(pos int, fn MFn) (int, error) {
	return m.LimitedLoop(pos, 0, -1, fn)
}

// must have n
func (m *Match) NLoop(pos int, n int, fn MFn) (int, error) {
	for i := 0; i < n; i++ {
		if p2, err := fn(pos); err != nil {
			return p2, err
		} else {
			pos = p2
		}
	}
	return pos, nil
}

//----------

func (m *Match) loopSep0(pos int, fn, sep MFn, lastSep bool) (int, error) {
	if m.sc.Reverse {
		i := 0
		return m.Loop(pos, m.W.And(
			func(p int) (int, error) {
				if i == 0 {
					if lastSep {
						return m.Optional(p, sep)
					}
					return p, nil
				}
				return sep(p)
			},
			fn,
			func(p int) (int, error) {
				i++
				return p, nil
			},
		))
	}

	i := 0
	done := false
	return m.Loop(pos, m.W.And(
		m.W.PtrFalse(&done),
		m.W.And(
			func(p int) (int, error) {
				if i == 0 {
					return p, nil
				}
				return sep(p)
			},
			m.W.Or(
				fn,
				func(p int) (int, error) {
					if i > 0 && lastSep {
						done = true
						return p, nil
					}
					return p, NoMatchErr
				},
			),
			func(p int) (int, error) { i++; return p, nil },
		),
	))
}
func (m *Match) LoopSep(pos int, fn, sep MFn) (int, error) {
	return m.loopSep0(pos, fn, sep, false)
}
func (m *Match) LoopSepCanHaveLast(pos int, fn, sep MFn) (int, error) {
	return m.loopSep0(pos, fn, sep, true)
}

//----------

func (m *Match) PtrFn(pos int, fn *MFn) (int, error) {
	return (*fn)(pos)
}

//---------- NOTE: not so "generic" util funcs (more specific)

func (m *Match) Spaces(pos int, includeNL bool, escape rune) (int, error) {
	valid := func(ru rune) bool {
		return unicode.IsSpace(ru) && (includeNL || ru != '\n')
	}
	return m.Loop(pos, m.W.Or(
		// escapes spaces // allow any space to be escaped
		m.W.And(
			m.W.StaticTrue(escape != 0),
			m.W.Rune(escape),
			m.W.RuneFn(unicode.IsSpace),
		),

		m.W.RuneFn(valid),
	))
}

//----------

func (m *Match) EscapeAny(pos int, escape rune) (int, error) {
	if escape == 0 {
		return pos, NoMatchErr
	}
	return m.AndR(pos,
		m.W.Rune(escape),
		m.OneRune,
	)
}

//----------

func (m *Match) ToNLOrErr(pos int, includeNL bool, esc rune) (int, error) {
	done := false
	valid := func(ru rune) bool {
		isNL := ru == '\n'
		if includeNL && isNL {
			done = true
			return true
		}
		return !isNL
	}
	return m.OptLoop(pos, m.W.And(
		m.W.PtrFalse(&done),
		m.W.Or(
			m.W.EscapeAny(esc),
			m.W.RuneFn(valid),
		)),
	)
}

//----------

// match opened/closed sections.
func (m *Match) Section(pos int, open, close string, esc rune, failOnNewline bool, maxLen int, eofClose bool, consumeFn MFn) (int, error) {
	if m.sc.Reverse {
		open, close = close, open
	}
	parseNL := func(pos int) (int, error) {
		return m.And(pos,
			m.W.StaticTrue(failOnNewline),
			m.W.Rune('\n'),
		)
	}
	parseClose := func(pos int) (int, error) {
		return m.Or(pos,
			m.W.Sequence(close),
			m.W.And(
				m.W.StaticTrue(eofClose),
				m.Eof,
			),
		)
	}
	return m.And(pos,
		m.W.Sequence(open),
		m.W.LimitedLoop(0, maxLen, m.W.And(
			m.W.MustErr(parseNL),
			m.W.MustErr(parseClose),
			m.W.Or(
				m.W.EscapeAny(esc),
				consumeFn,
			),
		)),
		parseClose,
	)
}

func (m *Match) StringSection(pos int, openclose string, esc rune, failOnNewline bool, maxLen int, eofClose bool) (int, error) {
	return m.Section(pos, openclose, openclose, esc, failOnNewline, maxLen, eofClose, m.OneRune)
}

//----------

func (m *Match) DoubleQuotedString(pos int, maxLen int) (int, error) {
	return m.StringSection(pos, "\"", '\\', true, maxLen, false)
}
func (m *Match) QuotedString(pos int) (int, error) {
	return m.QuotedString2(pos, '\\', 3000, 8)
}

// allows escaped runes with esc!=0
func (m *Match) QuotedString2(pos int, esc rune, maxLen1, maxLen2 int) (int, error) {
	return m.Or(pos,
		// doublequote: fail on newline, eof doesn't close
		m.W.StringSection("\"", esc, true, maxLen1, false),
		// singlequote: fail on newline, eof doesn't close (usually a smaller maxlen)
		m.W.StringSection("'", esc, true, maxLen2, false),
		// backquote: can have newline, eof doesn't close
		m.W.StringSection("`", esc, false, maxLen1, false),
	)
}

//----------

func (m *Match) Letter(pos int) (int, error) {
	return m.RuneFn(pos, unicode.IsLetter)
}
func (m *Match) Digit(pos int) (int, error) {
	return m.RuneFn(pos, unicode.IsDigit)
}
func (m *Match) Digits(pos int) (int, error) {
	return m.RuneFnLoop(pos, unicode.IsDigit)
}

func (m *Match) Integer(pos int) (int, error) {
	// TODO: reverse
	//u := "[+-]?[0-9]+"
	//return m.RegexpFromStartCached(u)

	return m.AndR(pos,
		m.W.Optional(m.sign),
		m.Digits,
	)
}
func (m *Match) sign(pos int) (int, error) {
	return m.RuneOneOf(pos, []rune("+-"))
}

func (m *Match) Float(pos int) (int, error) {
	// TODO: reverse
	//u := "[+-]?([0-9]*[.])?[0-9]+"
	//u := "[+-]?(\\d+([.]\\d*)?([eE][+-]?\\d+)?|[.]\\d+([eE][+-]?\\d+)?)"
	//return m.RegexpFromStartCached(u, 100)

	// -1.2
	// -1.2e3
	// .2
	// .2e3
	return m.AndR(pos,
		m.W.Optional(m.Integer),
		// fraction (must have)
		m.W.AndR(
			m.W.Rune('.'),
			m.Digits,
		),
		// exponent
		m.W.Optional(m.W.AndR(
			m.W.RuneOneOf([]rune("eE")),
			m.W.Optional(m.sign),
			m.Digits,
		)),
	)
}

//----------

func (m *Match) RegexpFromStart(pos int, res string, cache bool, maxLen int) (int, error) {
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
			return pos, err
		}
		re = re3
		if cache {
			m.cache.regexps[res] = re
		}
	}

	// limit input to be read
	src := m.sc.SrcFrom(pos)
	max := maxLen
	if max > len(src) {
		max = len(src)
	}
	src = m.sc.SrcFromTo(pos, pos+max)

	locs := re.FindIndex(src)
	if len(locs) == 0 {
		return pos, NoMatchErr
	}
	pos += locs[1]
	return pos, nil
}
func (m *Match) RegexpFromStartCached(pos int, res string, maxLen int) (int, error) {
	return m.RegexpFromStart(pos, res, true, maxLen)
}

//---------- NOTE: these util funcs don't affect the position

func (m *Match) MustErr(pos int, fn MFn) (int, error) {
	if _, err := fn(pos); err != nil {
		return pos, nil
	}
	return pos, NoMatchErr
}
func (m *Match) PtrTrue(pos int, v *bool) (int, error) {
	if *v {
		return pos, nil
	}
	return pos, NoMatchErr
}
func (m *Match) PtrFalse(pos int, v *bool) (int, error) {
	if !*v {
		return pos, nil
	}
	return pos, NoMatchErr
}
func (m *Match) StaticTrue(pos int, v bool) (int, error) {
	if v {
		return pos, nil
	}
	return pos, NoMatchErr
}
func (m *Match) StaticFalse(pos int, v bool) (int, error) {
	if !v {
		return pos, nil
	}
	return pos, NoMatchErr
}
func (m *Match) FnTrue(pos int, fn func() bool) (int, error) {
	if fn() {
		return pos, nil
	}
	return pos, NoMatchErr
}

//----------

func (m *Match) Eof(pos int) (int, error) {
	if _, _, err := m.sc.ReadRune(pos); err != nil {
		if m.sc.Reverse {
			if err == SOF {
				return pos, nil
			}
		} else {
			if err == EOF {
				return pos, nil
			}
		}
	}
	return pos, NoMatchErr
}
func (m *Match) NotEof(p int) (int, error) {
	return m.MustErr(p, m.Eof)
}

//----------

func (m *Match) ReverseMode(pos int, reverse bool, fn MFn) (int, error) {
	tmp := m.sc.Reverse
	m.sc.Reverse = reverse
	defer func() { m.sc.Reverse = tmp }()
	return fn(pos)
}

//---------- NOTE: value helpers

func (m *Match) OnValue(pos int, fn VFn, cb func(any)) (int, error) {
	if v, p2, err := fn(pos); err != nil {
		return p2, err
	} else {
		cb(v)
		return p2, nil
	}
}
func (m *Match) OnValue2(pos int, fn VFn, cb func(any) error) (int, error) {
	if v, p2, err := fn(pos); err != nil {
		return p2, err
	} else {
		err2 := cb(v)
		return p2, err2
	}
}
func (m *Match) OrValue(pos int, fns ...VFn) (any, int, error) {
	w := []MFn{}
	res := (any)(nil)
	for _, fn := range fns {
		fn2 := m.W.OnValue(fn, func(v any) { res = v })
		w = append(w, fn2)
	}
	if p2, err := m.Or(pos, w...); err != nil {
		return nil, p2, err
	} else {
		return res, p2, nil
	}
}
func (m *Match) BytesValue(pos int, fn MFn) (any, int, error) {
	if p2, err := fn(pos); err != nil {
		return nil, p2, err
	} else {
		src := m.sc.SrcFromTo(pos, p2)
		return src, p2, nil
	}
}
func (m *Match) StringValue(pos int, fn MFn) (any, int, error) {
	if v, p2, err := m.BytesValue(pos, fn); err != nil {
		return nil, p2, err
	} else {
		return string(v.([]byte)), p2, nil
	}
}
func (m *Match) RuneValue(pos int, fn MFn) (any, int, error) {
	if v, p2, err := m.BytesValue(pos, fn); err != nil {
		return nil, p2, err
	} else {
		rs := []rune(string(v.([]byte)))
		if len(rs) != 1 {
			return nil, pos, fmt.Errorf("expecting only one rune: %v", len(rs))
		}
		return rs[0], p2, nil
	}
}
func (m *Match) IntValue(pos int) (any, int, error) {
	if v, p2, err := m.StringValue(pos, m.Integer); err != nil {
		return nil, p2, err
	} else {
		if u, err := strconv.ParseInt(v.(string), 10, bits.UintSize); err != nil {
			return nil, pos, err
		} else {
			return int(u), p2, nil
		}
	}
}
func (m *Match) IntFnValue(pos int, fn MFn) (any, int, error) {
	if v, p2, err := m.StringValue(pos, fn); err != nil {
		return nil, p2, err
	} else {
		if u, err := strconv.ParseInt(v.(string), 10, bits.UintSize); err != nil {
			return nil, pos, err
		} else {
			return int(u), p2, nil
		}
	}
}
func (m *Match) Int64Value(pos int) (any, int, error) {
	if v, p2, err := m.StringValue(pos, m.Integer); err != nil {
		return nil, p2, err
	} else {
		if u, err := strconv.ParseInt(v.(string), 10, 64); err != nil {
			return nil, pos, err
		} else {
			return u, p2, nil
		}
	}
}
func (m *Match) Float64Value(pos int) (any, int, error) {
	if v, p2, err := m.StringValue(pos, m.Float); err != nil {
		return nil, p2, err
	} else {
		if u, err := strconv.ParseFloat(v.(string), 64); err != nil {
			return nil, pos, err
		} else {
			return u, p2, nil
		}
	}
}

//---------- NOTE: debug helpers

// useful in And's
func (m *Match) PrintfNoErr(pos int, f string, args ...any) (int, error) {
	fmt.Printf(f, args...)
	return pos, nil
}

// useful in Or's
func (m *Match) PrintfErr(pos int, f string, args ...any) (int, error) {
	fmt.Printf(f, args...)
	return pos, fmt.Errorf("printfErr")
}

//---------- NOTE: error helpers

func (m *Match) FatalOnError(pos int, s string, fn MFn) (int, error) {
	if p2, err := fn(pos); err != nil {
		return p2, m.sc.EnsureFatalError(err)
	} else {
		return p2, nil
	}
}

//----------
//----------
//----------
