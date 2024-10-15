package pscan

import (
	"fmt"
	"io"
	"math/bits"
	"regexp"
	"slices"
	"strconv"
	"strings"
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

func (m *Match) And(pos int, fns ...MFn) (int, error) {
	opt := AndOpt{}
	return m.And2(pos, opt, fns...)
}

// runs in order, even when in reverse mode; usefull because and() might be needed to run in order, but have each fn honor internally the reverse setting
func (m *Match) AndNoReverse(pos int, fns ...MFn) (int, error) {
	off := false // constant, never changes
	opt := AndOpt{Reverse: &off}
	return m.And2(pos, opt, fns...)
}

func (m *Match) AndOptSpaces(pos int, sopt SpacesOpt, fns ...MFn) (int, error) {
	aopt := AndOpt{OptSpaces: &sopt}
	return m.And2(pos, aopt, fns...)
}

func (m *Match) And2(pos int, aopt AndOpt, fns ...MFn) (int, error) {

	optSpacesFn := func(pos2 *int) {
		if aopt.OptSpaces == nil {
			return
		}
		p3, err := m.Optional(*pos2,
			m.W.Spaces(*aopt.OptSpaces),
		)
		if err != nil {
			return // ignore
		}
		*pos2 = p3
	}

	reverse := m.sc.Reverse
	if aopt.Reverse != nil {
		reverse = *aopt.Reverse
	}

	if reverse {
		for i := len(fns) - 1; i >= 0; i-- {
			fn := fns[i]
			optSpacesFn(&pos)
			if p2, err := fn(pos); err != nil {
				return p2, err
			} else {
				pos = p2
			}
		}
		optSpacesFn(&pos)
		return pos, nil
	} else {
		for _, fn := range fns {
			optSpacesFn(&pos)
			if p2, err := fn(pos); err != nil {
				return p2, err
			} else {
				pos = p2
			}
		}
		optSpacesFn(&pos)
		return pos, nil
	}
}

// runs in order, even when in reverse mode
func (m *Match) Or(pos int, fns ...MFn) (int, error) {
	err0 := (error)(nil)
	p0 := pos
	for _, fn := range fns {
		if p2, err := fn(pos); err != nil {
			if IsFatalError(err) {
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
	p2, err := fn(pos)
	if err != nil {
		if IsFatalError(err) {
			return p2, err
		}
		return pos, nil
	}
	return p2, nil
}

func (m *Match) Peek(pos int, fn MFn) (int, error) {
	//// no op in reverse
	//if m.sc.Reverse {
	//	return pos, nil
	//}

	// re-reverse and still peek
	if m.sc.Reverse {
		_, err := m.ReverseMode(pos, false, fn)
		return pos, err
	}

	_, err := fn(pos)
	return pos, err
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
	return m.LoopOneOrMore(pos, func(p2 int) (int, error) {
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
	return m.LoopN(pos, n, m.W.ByteFn(fn))
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
	return m.LoopOneOrMore(pos, func(p2 int) (int, error) {
		return m.RuneFn(p2, fn)
	})
}

func (m *Match) RuneOneOf(pos int, rs []rune) (int, error) {
	return m.RuneFn(pos, func(ru rune) bool {
		return slices.Contains(rs, ru)
	})
}

func (m *Match) RuneNoneOf(pos int, rs []rune) (int, error) {
	return m.RuneFn(pos, func(ru rune) bool {
		return !slices.Contains(rs, ru)
	})
}

func (m *Match) RuneSequence(pos int, seq []rune) (int, error) {
	for i, l := 0, len(seq); i < l; i++ {
		ru := seq[i]
		if m.sc.Reverse {
			ru = seq[l-1-i]
		}
		if p2, err := m.Rune(pos, ru); err != nil {

			if m.sc.Debug {
				fmt.Printf("sequence fail: %q\n", string(seq))
			}

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
	return m.LoopN(pos, n, m.W.RuneFn(fn))
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
	return m.RuneSequence(pos, []rune(seq))
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
func (m *Match) Loop(pos int, min, max int, fn MFn) (int, error) {
	for i := 0; ; i++ {
		if max >= 0 && i >= max {
			return pos, nil
		}
		p2, err := fn(pos)
		if err != nil {
			if IsFatalError(err) {
				return p2, err
			}
			if i >= min {
				if m.sc.Debug {
					fmt.Println(m.sc.SrcError(p2, err))
				}

				return pos, nil // last good fn() position
			}
			return p2, err
		}
		pos = p2
	}
	return pos, nil
}

func (m *Match) LoopOneOrMore(pos int, fn MFn) (int, error) {
	return m.Loop(pos, 1, -1, fn)
}
func (m *Match) LoopZeroOrMore(pos int, fn MFn) (int, error) {
	return m.Loop(pos, 0, -1, fn)
}

// must have n
func (m *Match) LoopN(pos int, n int, fn MFn) (int, error) {
	return m.Loop(pos, n, n, fn)
}

//----------

func (m *Match) LoopSep(pos int, optLastSep bool, fn, sepFn MFn) (int, error) {
	i := 0
	pos2, err := m.LoopOneOrMore(pos, m.W.AndNoReverse(
		func(p int) (int, error) {
			if m.sc.Reverse {
				if i == 0 {
					if optLastSep {
						return m.Optional(p, sepFn)
					}
					return p, nil
				}
				return sepFn(p)
			} else {
				if i > 0 {
					return sepFn(p)
				}
				return p, nil
			}
		},
		fn,
		func(p int) (int, error) { i++; return p, nil },
	))
	if err != nil {
		return pos2, err
	}

	// try to consume last separator
	if !m.sc.Reverse {
		if optLastSep {
			pos3, err := m.Optional(pos2, sepFn)
			if err != nil { // ex: fatal err
				return pos3, err
			}
			pos2 = pos3
		}
	}

	return pos2, nil
}

//----------

func (m *Match) LoopStartEnd(pos int, min, max int, startFn, consumeFn, endFn MFn) (int, error) {

	// handling startFn=nil allows the reverse to work
	acceptNil := func(fn MFn) MFn {
		if fn == nil {
			return func(p int) (int, error) { return p, nil }
		}
		return fn
	}

	parseEnding := func(pos int) (int, error) {
		return m.StaticCondFn(pos, m.sc.Reverse, startFn, endFn)
	}
	return m.And(pos,
		acceptNil(startFn),
		m.W.Loop(min, max, m.W.AndNoReverse(
			m.W.MustErr(parseEnding),
			consumeFn,
		)),
		acceptNil(endFn),
	)
}

// note that it can read zero if on eof; inside a loop could give endless loop
func (m *Match) LoopUntilNLOrEof(pos int, max int, includeNL bool, esc rune) (int, error) {
	return m.LoopStartEnd(pos, 0, max,
		nil,
		m.W.Or(
			m.W.EscapeAny(esc),
			m.OneByte,
		),
		m.W.Or(
			m.W.StaticCondFn(includeNL,
				m.Newline,
				m.W.Peek(m.Newline),
			),
			m.Eof,
		),
	)
}

//----------

func (m *Match) PtrFn(pos int, fn *MFn) (int, error) {
	return (*fn)(pos)
}

//---------- NOTE: not so "generic" util funcs (more specific)

func (m *Match) Newline(pos int) (int, error) {
	return m.Rune(pos, '\n')
}
func (m *Match) Spaces(pos int, opt SpacesOpt) (int, error) {

	// escapes spaces // allow any space to be escaped
	escFn := func(p int) (int, error) { return p, io.EOF } // just a fail func
	if opt.HasEscape() {
		escFn = m.W.And(
			m.W.Rune(opt.Esc),
			m.W.RuneFn(unicode.IsSpace),
		)
	}

	return m.LoopOneOrMore(pos, m.W.Or(
		//m.W.AndNoReverse(
		//	m.W.StaticTrue(escape != 0),
		//	m.W.And(
		//		m.W.Rune(escape),
		//		m.W.RuneFn(unicode.IsSpace),
		//	),
		//),
		escFn,

		m.W.RuneFn(func(ru rune) bool {
			return unicode.IsSpace(ru) && (opt.IncludeNL || ru != '\n')
		}),
	))
}
func (m *Match) SpacesExceptNewline(pos int) (int, error) {
	return m.Spaces(pos, SpacesOpt{false, 0})
}
func (m *Match) SpacesIncludingNewline(pos int) (int, error) {
	return m.Spaces(pos, SpacesOpt{true, 0})
}

//----------

func (m *Match) EmptyLine(p int) (int, error) {
	return m.And(p,
		m.W.Optional(m.SpacesExceptNewline),
		m.Newline,
	)
}

// NOTE: because of eof, if inside a loop can lead to inf loop
func (m *Match) EmptyEof(p int) (int, error) {
	return m.And(p,
		m.W.Optional(m.SpacesExceptNewline),
		m.Eof,
	)
}

// NOTE: because of eof, if inside a loop can lead to inf loop
func (m *Match) EmptyRestOfLine(p int) (int, error) {
	return m.And(p,
		m.W.Optional(m.SpacesExceptNewline),
		// match end
		m.W.Or(
			m.Newline,
			m.Eof,
		),
	)
}

//----------

func (m *Match) EscapeAny(pos int, escape rune) (int, error) {
	if escape == 0 {
		return pos, NoMatchErr
	}
	return m.And(pos,
		m.W.Rune(escape),
		m.OneRune,
	)
}

//----------

//// match opened/closed sections.
//func (m *Match) Section(pos int, open, close string, esc rune, failOnNewline bool, max int, eofClose bool, consumeFn MFn) (int, error) {
//	if m.sc.Reverse {
//		open, close = close, open
//	}
//	parseNL := func(pos int) (int, error) {
//		return m.TrueFn(pos, failOnNewline, m.W.Rune('\n'))
//	}
//	parseClose := func(pos int) (int, error) {
//		return m.Or(pos,
//			m.W.Sequence(close),
//			m.W.TrueFn(eofClose, m.Eof),
//		)
//	}
//	return m.AndNoReverse(pos,
//		m.W.Sequence(open),
//		m.W.Loop(0, max, m.W.AndNoReverse(
//			m.W.MustErr(parseNL),
//			m.W.MustErr(parseClose),
//			m.W.Or(
//				m.W.EscapeAny(esc),
//				consumeFn,
//			),
//		)),
//		parseClose,
//	)
//}

// match opened/closed sections.
func (m *Match) Section(pos int, open, close string, esc rune, failOnNewline bool, max int, eofClose bool, consumeFn MFn) (int, error) {
	parseNL := func(pos int) (int, error) {
		return m.StaticCondFn(pos, failOnNewline, m.W.Rune('\n'), nil)
	}
	return m.And(pos,
		m.W.LoopStartEnd(0, max,
			// start
			m.W.Sequence(open),
			// consume
			m.W.AndNoReverse(
				m.W.MustErr(parseNL),
				m.W.Or(
					m.W.EscapeAny(esc),
					consumeFn,
				),
			),
			// end
			m.W.Or(
				m.W.Sequence(close),
				m.W.StaticCondFn(eofClose, m.Eof, nil),
			),
		),
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
	return m.LoopOneOrMore(pos, m.Digit)
}

func (m *Match) Integer(pos int) (int, error) {
	// TODO: reverse
	//u := "[+-]?[0-9]+"
	//return m.RegexpFromStartCached(u)

	return m.And(pos,
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
	return m.And(pos,
		m.W.Optional(m.Integer),
		// fraction (must have)
		m.W.And(
			m.W.Rune('.'),
			m.Digits,
		),
		// exponent
		m.W.Optional(m.W.And(
			m.W.RuneOneOf([]rune("eE")),
			m.W.Optional(m.sign),
			m.Digits,
		)),
	)
}

func (m *Match) FloatOrInteger(pos int) (int, error) {
	return m.Or(pos, m.Float, m.Integer)
}

func (m *Match) Identifier(pos int) (int, error) {
	return m.And(pos,
		m.W.RuneFn(func(ru rune) bool {
			return unicode.IsLetter(ru) ||
				strings.Contains("_", string(ru))
		}),
		m.W.Optional(m.W.RuneFnLoop(func(ru rune) bool {
			return unicode.IsLetter(ru) ||
				unicode.IsDigit(ru) ||
				strings.Contains("_", string(ru))
		})),
	)
}

func (m *Match) LettersAndDigits(pos int) (int, error) {
	return m.RuneFnLoop(pos, func(ru rune) bool {
		return unicode.IsLetter(ru) ||
			unicode.IsDigit(ru)
	})
}

func (m *Match) HexBytes(pos int) (int, error) {
	return m.ByteFnLoop(pos, func(b byte) bool {
		return (b >= '0' && b <= '9') ||
			(b >= 'a' && b <= 'f') ||
			(b >= 'A' && b <= 'F')
	})
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

// NOTE: if using with And(), consider using AndNoReverse()
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

func (m *Match) StaticCondFn(pos int, v bool, tfn, ffn MFn) (int, error) {
	if v {
		if tfn == nil {
			return pos, NoMatchErr
		}
		return tfn(pos)
	} else {
		if ffn == nil {
			return pos, NoMatchErr
		}
		return ffn(pos)
	}
}

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

func (m *Match) ReverseMode(pos int, reverse bool, fn MFn) (int, error) {
	restore := m.sc.Reverse
	m.sc.Reverse = reverse
	defer func() { m.sc.Reverse = restore }()
	return fn(pos)
}

//---------- NOTE: loop value helpers

func (m *Match) LoopValue(pos int, min, max int, fn VFn) (any, int, error) {
	return LoopValue[any](m, pos, min, max, fn)
}

func LoopValue[T any](m *Match, pos int, min, max int, fn VFn) (any, int, error) {
	w := []T{}
	pos4, err := m.Loop(pos, min, max, func(pos2 int) (int, error) {
		v, pos3, err := fn(pos2)
		if err != nil {
			return pos3, err
		}
		w = append(w, v.(T))
		return pos3, nil
	})
	return w, pos4, err
}
func WLoopValue[T any](w *Wrap, min, max int, fn VFn) VFn {
	return func(pos int) (any, int, error) {
		return LoopValue[T](w.M, pos, min, max, fn)
	}
}

func (m *Match) LoopSepValue(pos int, optLastSep bool, fn VFn, sepFn MFn) (any, int, error) {
	return LoopSepValue[any](m.sc, pos, optLastSep, fn, sepFn)
}

func LoopSepValue[T any](sc *Scanner, pos int, optLastSep bool, fn VFn, sepFn MFn) (any, int, error) {
	w := []T{}
	pos4, err := sc.M.LoopSep(pos,
		optLastSep,
		func(pos2 int) (int, error) {
			v, pos3, err := fn(pos2)
			if err != nil {
				return pos3, err
			}
			w = append(w, v.(T))
			return pos3, nil
		},
		sepFn,
	)
	return w, pos4, err
}
func WLoopSepValue[T any](sc *Scanner, optLastSep bool, fn VFn, sepFn MFn) VFn {
	return func(pos int) (any, int, error) {
		return LoopSepValue[T](sc, pos, optLastSep, fn, sepFn)
	}
}

//---------- NOTE: value helpers

func OnValueM[T any](pos int, fn VFn, cb func(T) error) (int, error) {
	_, p2, err := OnValueV[T](pos, fn, func(v T) (any, error) {
		return nil, cb(v)
	})
	return p2, err
}
func WOnValueM[T any](fn VFn, cb func(T) error) MFn {
	return func(pos int) (int, error) {
		return OnValueM[T](pos, fn, cb)
	}
}

func OnValueV[T any](pos int, fn VFn, cb func(T) (any, error)) (any, int, error) {
	v2, p2, err := fn(pos)
	if err != nil {
		return nil, p2, err
	}
	v3, ok := v2.(T)
	if !ok {
		var zero T
		err := fmt.Errorf("pscan.onvalue: type is %T, not %T", v2, zero)
		return nil, p2, FatalError(err)
	}
	v4, err4 := cb(v3)
	return v4, p2, err4
}
func WOnValueV[T any](fn VFn, cb func(T) (any, error)) VFn {
	return func(pos int) (any, int, error) {
		return OnValueV[T](pos, fn, cb)
	}
}

func (m *Match) AndValue(pos int, fns ...VFn) (any, int, error) {
	// build funcs that keep the values
	res := []any{}
	fns2 := []MFn{}
	for _, fn := range fns {
		fn2 := WOnValueM(fn, func(v any) error {
			res = append(res, v)
			return nil
		})
		fns2 = append(fns2, fn2)
	}

	p2, err := m.And(pos, fns2...)
	if err != nil {
		return nil, p2, err
	}

	// TODO: reverse? reverse res?

	return res, p2, nil
}

func (m *Match) AndFlexValue(pos int, fns ...any) (any, int, error) {
	// build funcs that keep the values in res
	res := []any{}
	fns2 := []MFn{}
	valFnCount := 0
	for _, fn := range fns {
		switch t := fn.(type) {
		case VFn:
			valFnCount++
			mfn := func(pos int) (int, error) {
				v, p2, err := t(pos)
				if err != nil {
					return p2, err
				}
				res = append(res, v)
				return p2, nil
			}
			fns2 = append(fns2, mfn)
		case MFn:
			fns2 = append(fns2, t)
		default:
			err := fmt.Errorf("unexpected type: %T", fn)
			panic(err)
		}
	}
	if valFnCount == 0 {
		panic(fmt.Errorf("missing value funcs"))
	}

	p2, err := m.And(pos, fns2...)
	if err != nil {
		return nil, p2, err
	}
	if len(res) == 1 {
		return res[0], p2, nil
	}
	return res, p2, nil
}

func (m *Match) OrValue(pos int, fns ...VFn) (any, int, error) {
	// build funcs that keep the value
	fns2 := []MFn{}
	res := (any)(nil)
	for _, fn := range fns {
		fn2 := WOnValueM(fn, func(v any) error {
			res = v
			return nil
		})
		fns2 = append(fns2, fn2)
	}

	if p2, err := m.Or(pos, fns2...); err != nil {
		return nil, p2, err
	} else {
		return res, p2, nil
	}
}

func (m *Match) OptionalValue(pos int, fn VFn) (any, int, error) {
	v, p2, err := fn(pos)
	if err != nil {
		if IsFatalError(err) {
			return nil, p2, err
		}
		return nil, pos, nil
	}
	return v, p2, nil
}

func (m *Match) NilValue(pos int, fn MFn) (any, int, error) {
	p2, err := fn(pos)
	return nil, p2, err
}

func (m *Match) BytesValue(pos int, fn MFn) (any, int, error) {
	if p2, err := fn(pos); err != nil {
		return nil, p2, err
	} else {
		src := m.sc.SrcFromTo(pos, p2)
		return src, p2, nil
	}
}
func (m *Match) StrValue(pos int, fn MFn) (any, int, error) {
	if v, p2, err := m.BytesValue(pos, fn); err != nil {
		return nil, p2, err
	} else {
		return string(v.([]byte)), p2, nil
	}
}

func TStrValue[T ~string](sc *Scanner, pos int, fn MFn) (any, int, error) {
	v, p2, err := sc.M.StrValue(pos, fn)
	if err != nil {
		return nil, p2, err
	}
	return T(v.(string)), p2, err
}
func WTStrValue[T ~string](sc *Scanner, fn MFn) VFn {
	return func(pos int) (any, int, error) {
		return TStrValue[T](sc, pos, fn)
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
	if v, p2, err := m.StrValue(pos, m.Integer); err != nil {
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
	if v, p2, err := m.StrValue(pos, fn); err != nil {
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
	if v, p2, err := m.StrValue(pos, m.Integer); err != nil {
		return nil, p2, err
	} else {
		if u, err := strconv.ParseInt(v.(string), 10, 64); err != nil {
			return nil, pos, err
		} else {
			return u, p2, nil
		}
	}
}
func (m *Match) Float32Value(pos int) (any, int, error) {
	if v, p2, err := m.StrValue(pos, m.Float); err != nil {
		return nil, p2, err
	} else {
		if u, err := strconv.ParseFloat(v.(string), 32); err != nil {
			return nil, pos, err
		} else {
			return float32(u), p2, nil
		}
	}
}
func (m *Match) Float64Value(pos int) (any, int, error) {
	if v, p2, err := m.StrValue(pos, m.Float); err != nil {
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

func (m *Match) Printf(pos int, f string, args ...any) (int, error) {
	fmt.Printf(f, args...)
	return pos, nil
}
func (m *Match) PrintfForOr(pos int, f string, args ...any) (int, error) {
	return m.FailForOr(pos, func(p2 int) (int, error) {
		return m.Printf(pos, f, args...)
	})
}

func (m *Match) PrintLineColAndSrc(pos int) (int, error) {
	lc := ""
	if l, c, ok := m.sc.SrcLineCol(pos); ok {
		lc = fmt.Sprintf("%v:%v: ", l, c)
	}
	fmt.Printf("%v%q\n", lc, m.sc.SrcSection(pos))
	return pos, nil
}
func (m *Match) PrintLineColAndSrcForOr(pos int) (int, error) {
	return m.FailForOr(pos, m.PrintLineColAndSrc)
}

func (m *Match) PrintPosAndSrc(pos int) (int, error) {
	fmt.Printf("%v: %q\n", pos, m.sc.SrcSection(pos))
	return pos, nil
}
func (m *Match) PrintPosAndSrcForOr(pos int) (int, error) {
	return m.FailForOr(pos, m.PrintPosAndSrc)
}

//---------- NOTE: error helpers

func (m *Match) FatalOnError(pos int, s string, fn MFn) (int, error) {
	p2, err := fn(pos)
	if err != nil {
		if s != "" {
			err = fmt.Errorf("%v: %w", s, err)
		}
		return p2, FatalError(err)
	}
	return p2, nil
}
func (m *Match) FailForOr(pos int, fn MFn) (int, error) {
	p2, err := fn(pos)
	if err != nil {
		return p2, err
	}
	return p2, fmt.Errorf("fail for or")
}

//----------
//----------
//----------
