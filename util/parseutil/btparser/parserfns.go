package btparser

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

func (p *Parser) mvByte(pos Pos) (byte, MPos, error) {
	if pos > p.farthest {
		p.farthest = pos
	}

	l := Pos(len(p.src))
	if pos >= l {
		return 0, MPos{l, l}, NoMatchErr
	}
	b := p.src[pos]
	p2 := pos + 1
	return b, MPos{pos, p2}, nil
}
func (p *Parser) mvRune(pos Pos) (rune, MPos, error) {
	if pos > p.farthest {
		p.farthest = pos
	}

	ru, size := utf8.DecodeRune(p.src[pos:])
	if size == 0 {
		return 0, MPos{pos, pos}, NoMatchErr
	}
	p2 := pos + Pos(size)
	return ru, MPos{pos, p2}, nil
}
func (p *Parser) mvLastRune(pos Pos) (rune, MPos, error) {
	ru, size := utf8.DecodeLastRune(p.src[:pos])
	if size == 0 {
		return 0, MPos{pos, pos}, NoMatchErr
	}
	p2 := pos - Pos(size)
	return ru, MPos{p2, pos}, nil
}

//----------

// there can be no nested tokens; should be set at leaf nodes
func (p *Parser) mToken(pos Pos, fn MFn) (MPos, error) {
	pos = p.runIgnore(pos)

	p.tokenC++
	defer func() { p.tokenC-- }()
	if p.tokenC > 1 {
		err := fmt.Errorf("nested tokens: %v", p.Snippet(MPos{pos, pos}))
		panic(err)
	}

	return fn(pos)
}

//----------

func (p *Parser) mAnd(pos Pos, fns ...MFn) (MPos, error) {
	p0 := pos
	for _, fn := range fns {
		if mp, err := fn(pos); err != nil {
			return mp, err
		} else {
			pos = mp.End
		}
	}
	return MPos{p0, pos}, nil
}
func (p *Parser) mOr(pos Pos, fns ...MFn) (MPos, error) {
	for _, fn := range fns {
		if mp, err := fn(pos); err != nil {
			if IsFatalError(err) {
				return mp, err
			}
		} else {
			return mp, nil
		}
	}
	return MPos{pos, pos}, NoMatchErr
}

func (p *Parser) mOptional(pos Pos, fn MFn) (MPos, error) {
	if mp, err := fn(pos); err != nil {
		if IsFatalError(err) {
			return mp, err
		}
		return MPos{pos, pos}, nil
	} else {
		return mp, nil
	}
}
func (p *Parser) mLookahead(pos Pos, fn MFn) (MPos, error) {
	if mp, err := fn(pos); err != nil {
		return mp, err
	} else {
		return MPos{pos, pos}, nil // stay in same pos
	}
}
func (p *Parser) mLookbackN(pos Pos, n int, fn MFn) (MPos, error) {
	p2 := pos - Pos(n)
	if p2 < 0 {
		return MPos{pos, pos}, NoMatchErr
	}
	if mp, err := fn(p2); err != nil {
		return mp, err
	}
	return MPos{pos, pos}, nil // stay in same pos
}

//----------

func (p *Parser) mNot(pos Pos, fn MFn) (MPos, error) {
	if mp, err := fn(pos); err != nil {
		return MPos{pos, pos}, nil
	} else {
		return mp, NoMatchErr
	}
}
func (p *Parser) mFail(pos Pos) (MPos, error) {
	return MPos{pos, pos}, NoMatchErr
}
func (p *Parser) mNoOp(pos Pos) (MPos, error) {
	return MPos{pos, pos}, nil
}
func (p *Parser) mNoOp2(pos Pos, fn func() error) (MPos, error) {
	return MPos{pos, pos}, fn()
}
func (p *Parser) mEof(pos Pos) (MPos, error) {
	if _, mp, err := p.mvByte(pos); err != nil { // eof
		return MPos{pos, pos}, nil
	} else {
		return mp, NoMatchErr
	}
}

// start-of-file
func (p *Parser) mSof(pos Pos) (MPos, error) {
	if pos == 0 {
		return MPos{pos, pos}, nil
	}
	return MPos{pos, pos}, NoMatchErr
}

//----------

func (p *Parser) mByteFn(pos Pos, fn func(byte) bool) (MPos, error) {
	return mHandleVFn(pos, p.mvByte, BoolErrFn(fn))
}
func (p *Parser) mByteFnLoop(pos Pos, fn func(byte) bool) (MPos, error) {
	return p.mLoop1(pos, p.byteFn(fn))
}
func (p *Parser) mByte(pos Pos, b byte) (MPos, error) {
	return p.mByteFn(pos, func(b2 byte) bool { return b2 == b })
}

//----------

func (p *Parser) mRuneFn(pos Pos, fn func(rune) bool) (MPos, error) {
	return mHandleVFn(pos, p.mvRune, BoolErrFn(fn))
}
func (p *Parser) mRuneFnLoop(pos Pos, fn func(rune) bool) (MPos, error) {
	return p.mLoop1(pos, p.runeFn(fn))
}
func (p *Parser) mRune(pos Pos, ru rune) (MPos, error) {
	return p.mRuneFn(pos, func(ru2 rune) bool { return ru2 == ru })
}
func (p *Parser) mContainsRune(pos Pos, s string) (MPos, error) {
	return p.mRuneFn(pos, func(ru rune) bool {
		return strings.ContainsRune(s, ru)
	})
}
func (p *Parser) mContainsRune2(pos Pos, rus []rune) (MPos, error) {
	return p.mRuneFn(pos, func(ru rune) bool {
		return slices.Contains(rus, ru)
	})
}
func (p *Parser) mAnyRune(pos Pos) (MPos, error) {
	_, mp, err := p.mvRune(pos)
	return mp, err
}
func (p *Parser) mNRunes(pos Pos, n int) (MPos, error) {
	p0 := pos
	for k := 0; k < n; k++ {
		if _, mp, err := p.mvRune(pos); err != nil {
			return MPos{p0, pos}, err
		} else {
			pos = mp.End
		}
	}
	return MPos{p0, pos}, nil
}
func (p *Parser) mMaxNRunes(pos Pos, n int) (MPos, error) {
	p0 := pos
	for k := 0; k < n; k++ {
		if _, mp, err := p.mvRune(pos); err != nil {
			if k == 0 {
				return MPos{p0, pos}, err
			}
			return MPos{p0, pos}, nil
		} else {
			pos = mp.End
		}
	}
	return MPos{p0, pos}, nil
}
func (p *Parser) mSeq(pos Pos, s string) (MPos, error) {
	if s == "" {
		return p.mFail(pos)
	}
	p0 := pos
	for _, ru := range s {
		if ru2, mp, err := p.mvRune(pos); err != nil {
			return MPos{p0, pos}, err
		} else if ru2 != ru {
			return MPos{p0, pos}, NoMatchErr
		} else {
			pos = mp.End
		}
	}
	return MPos{p0, pos}, nil
}

//----------

func (p *Parser) mLoop1(pos Pos, fn MFn) (MPos, error) {
	p0 := pos
	for k := 0; ; k++ {
		if mp, err := fn(pos); err != nil {
			if IsFatalError(err) {
				return mp, err
			}
			if k == 0 {
				return mp, err
			}
			return MPos{p0, pos}, nil
		} else {
			if mp.End == pos {
				return mp, p.loopNoProgressError("mLoop1", pos)
			}
			pos = mp.End
		}
	}
}

// TODO: other loops to accept this "looper" as arg
func (p *Parser) mLoop2(pos Pos, minN, maxN int, fn MFn) (MPos, error) {
	p0 := pos
	for k := 0; maxN < 0 || k < maxN; k++ {
		if mp, err := fn(pos); err != nil {
			if k == 0 && minN == 0 { // works like an optional
				return MPos{p0, p0}, nil
			}
			if minN > 0 && k < minN { // didn't reach required n
				return MPos{p0, pos}, NoMatchErr
			}
			return MPos{p0, pos}, nil
		} else {
			if mp.End == pos {
				return mp, p.loopNoProgressError("mLoop2", pos)
			}
			pos = mp.End
		}
	}
	return MPos{p0, pos}, nil
}

func (p *Parser) mLoopSep(pos Pos, optLastSep bool, fn, sepFn MFn) (MPos, error) {
	p0 := pos
	for k := 0; ; k++ {
		// separator
		p2 := pos
		if k > 0 {
			if mp, err := sepFn(pos); err != nil {
				return MPos{p0, pos}, nil
			} else {
				if mp.End == pos {
					return mp, p.loopNoProgressError("mLoopSep.sep", pos)
				}
				pos = mp.End
			}
		}

		if mp, err := fn(pos); err != nil {
			if IsFatalError(err) {
				return mp, err
			}
			if k == 0 {
				return MPos{p0, pos}, err
			}
			if k > 0 && !optLastSep {
				return MPos{p0, p2}, nil
			}
			return MPos{p0, pos}, nil
		} else {
			if mp.End == pos {
				return mp, p.loopNoProgressError("mLoopSep.fn", pos)
			}
			pos = mp.End
		}
	}
}
func (p *Parser) mLoopStartEnd(pos Pos, startFn, consumeFn, endFn MFn) (MPos, error) {
	if startFn == nil {
		startFn = p.mNoOp
	}
	return p.mAnd(pos,
		startFn,
		p.optional(p.loop1(p.and(
			p.not(endFn),
			consumeFn,
		))),
		endFn,
	)
}

func (p *Parser) mLoopToNLOrEof(pos Pos, esc rune, includeNL bool) (MPos, error) {
	nlFn := (MFn)(nil)
	if includeNL {
		nlFn = p.mNewline
	} else {
		nlFn = p.lookahead(p.mNewline)
	}
	return p.mLoopStartEnd(pos,
		nil,
		p.or(
			p.escape(esc),
			p.mAnyRune,
		),
		p.or(
			nlFn,
			p.mEof,
		),
	)
}

//----------

func (p *Parser) mLetter(pos Pos) (MPos, error) {
	return p.mRuneFn(pos, unicode.IsLetter)
}
func (p *Parser) mDigit(pos Pos) (MPos, error) {
	return p.mRuneFn(pos, unicode.IsDigit)
}
func (p *Parser) mDigitNotZero(pos Pos) (MPos, error) {
	return p.mRuneFn(pos, func(ru rune) bool {
		return ru != '0' && unicode.IsDigit(ru)
	})
}
func (p *Parser) mDigits(pos Pos) (MPos, error) {
	return p.mLoop1(pos, p.mDigit)
}

func (p *Parser) mFloat(pos Pos) (MPos, error) {
	return p.mFloat2(pos, '.')
}
func (p *Parser) mFloat2(pos Pos, sep rune) (MPos, error) {
	return p.mAnd(pos,
		//p.WOptional(p.MInteger), // wrong, won't allow "-0.1"

		p.optional(p.mSign),
		p.or(
			p.and(
				p.rune('0'),
				// avoid 2nd zero
				p.optional(p.and(
					p.mDigitNotZero,
					p.optional(p.mDigits),
				)),
			),
			p.and(
				p.mDigitNotZero,
				p.optional(p.mDigits),
			),
		),

		// fraction
		p.and(
			p.rune(sep),
			p.mDigits,
		),

		// TODO: exponent?
	)
}
func (p *Parser) mInteger(pos Pos) (MPos, error) {
	return p.mOr(pos,
		p.and(
			p.optional(p.mSign),
			p.mDigitNotZero,
			p.optional(p.mDigits),
		),
		// just zero
		p.and(
			p.rune('0'),
			p.lookahead(p.not(p.mDigits)),
		),
	)
}
func (p *Parser) mSign(pos Pos) (MPos, error) {
	return p.mContainsRune(pos, "-+")
}
func (p *Parser) mBool(pos Pos) (MPos, error) {
	return p.mOr(pos,
		p.seq("true"), p.seq("false"),
		p.seq("True"), p.seq("False"),
		p.seq("TRUE"), p.seq("FALSE"),
	)
}
func (p *Parser) mHexBytes(pos Pos) (MPos, error) {
	return p.mByteFnLoop(pos, func(b byte) bool {
		return (b >= '0' && b <= '9') ||
			(b >= 'a' && b <= 'f') ||
			(b >= 'A' && b <= 'F')
	})
}

//----------

// NOTE: use p.MVTime
//// TODO: this a simple/fixed/rigid time, needs improvement
//// TODO: ex: fmt "_2" won't match "2"
//func (p *Parser) MTime(pos Pos, fmt string) (MPos, error) {
//	p0 := pos
//	for _, ru := range fmt {
//		switch {
//		case unicode.IsDigit(ru):
//			mp, err := p.MDigit(pos)
//			if err != nil {
//				return mp, err
//			}
//			pos = mp.End

//		case unicode.IsLetter(ru):
//			// ex: Jan/monday/pm/am: time/format.go
//			mp, err := p.MRuneFn(pos, unicode.IsLetter)
//			if err != nil {
//				return mp, err
//			}
//			pos = mp.End

//		default: // match "-", "/", ...
//			mp, err := p.MRune(pos, ru)
//			if err != nil {
//				return mp, err
//			}
//			pos = mp.End
//		}
//	}
//	return MPos{p0, pos}, nil
//}

//----------

func (p *Parser) mSpace(pos Pos) (MPos, error) {
	return p.mRuneFn(pos, unicode.IsSpace)
}
func (p *Parser) mSpaces(pos Pos) (MPos, error) {
	return p.mRuneFnLoop(pos, unicode.IsSpace)
}
func (p *Parser) mSpacesExceptNewline(pos Pos) (MPos, error) {
	return p.mRuneFnLoop(pos, func(ru rune) bool {
		return ru != '\n' && unicode.IsSpace(ru)
	})
}
func (p *Parser) mNewline(pos Pos) (MPos, error) {
	return p.mRune(pos, '\n')
}

//----------

// the ignore fn should not consume newlines
func (p *Parser) mEmptyLinesExceptNewline(pos Pos, ignore MFn) (MPos, error) {
	ignores := p.loop1(ignore)
	return p.mOr(pos,
		// start of file (special case): also consumes ending newlines
		p.and(
			p.mSof,
			p.loop1(p.or(
				ignores,
				p.mNewline,
			)),
		),
		// middle of file
		p.loop1(p.or(
			ignores,
			// empty lines
			p.and(
				p.mNewline,
				p.optional(ignores),
				p.lookahead(p.mNewline),
			),
		)),
	)
}

//----------

func (p *Parser) mIdentifier(pos Pos) (MPos, error) {
	return p.mAnd(pos,
		p.runeFn(func(ru rune) bool {
			return unicode.IsLetter(ru) || ru == '_'
		}),
		p.optional(p.runeFnLoop(func(ru rune) bool {
			return unicode.IsLetter(ru) || unicode.IsDigit(ru) || ru == '_'
		})),
	)
}
func (p *Parser) mEscape(pos Pos, esc rune) (MPos, error) {
	if esc == 0 {
		return p.mFail(pos)
	}
	return p.mAnd(pos,
		p.rune(esc),
		p.nRunes(1),
	)
}
func (p *Parser) mAnyExceptNewline(pos Pos) (MPos, error) {
	return p.mAnd(pos,
		p.not(p.rune('\n')),
		p.mAnyRune,
	)
}

//----------

func (p *Parser) mQuotedString1(pos Pos) (MPos, error) {
	return p.mSection(pos, "\"", "\"", '\\', false, false, p.mAnyExceptNewline)
}
func (p *Parser) mLineComment1(pos Pos, open string) (MPos, error) {
	return p.mSection(pos, open, "", 0, true, true, p.mAnyExceptNewline)
}

func (p *Parser) mSection(pos Pos,
	open, close string,
	esc rune,
	newlineCloses, eofCloses bool,
	consume MFn) (MPos, error) {

	closeFn := p.seq(close)
	if eofCloses {
		closeFn = p.or(
			p.mEof,
			closeFn,
		)
	}
	if newlineCloses {
		closeFn = p.or(
			p.lookahead(p.rune('\n')), // don't consume
			closeFn,
		)
	}
	return p.mLoopStartEnd(pos,
		p.seq(open),
		p.or(
			p.escape(esc),
			consume,
		),
		closeFn,
	)
}

//----------
//----------
//----------

func mHandleMFn[T any](pos Pos, fn1 MFn, fn2 VMaker[T]) (T, MPos, error) {
	if mp, err := fn1(pos); err != nil {
		var zero T
		return zero, mp, err
	} else {
		v, err := fn2(mp)
		return v, mp, err
	}
}
func mHandleVFn[T any](pos Pos, fn1 VFn[T], fn2 VHandler[T]) (MPos, error) {
	if v, mp, err := fn1(pos); err != nil {
		return mp, err
	} else {
		return mp, fn2(v)
	}
}

//----------

func (p *Parser) mvSource(pos Pos, fn MFn) ([]byte, MPos, error) {
	return mHandleMFn(pos, fn, func(mp MPos) ([]byte, error) {
		return p.Source(mp), nil
	})
}
func (p *Parser) mvSourceStr(pos Pos, fn MFn) (string, MPos, error) {
	b, mp, err := p.mvSource(pos, fn)
	return string(b), mp, err
}

func (p *Parser) mvFloat(pos Pos) (float64, MPos, error) {
	return mHandleMFn(pos, p.mFloat, func(mp MPos) (float64, error) {
		return strconv.ParseFloat(p.SourceStr(mp), 64)
	})
}
func (p *Parser) mvInteger(pos Pos) (int, MPos, error) {
	return mHandleMFn(pos, p.mInteger, func(mp MPos) (int, error) {
		v, err := strconv.ParseInt(p.SourceStr(mp), 10, 64)
		return int(v), err
	})
}
func (p *Parser) mvBool(pos Pos) (bool, MPos, error) {
	return mHandleMFn(pos, p.mBool, func(mp MPos) (bool, error) {
		return strconv.ParseBool(p.SourceStr(mp))
	})
}
func (p *Parser) mvIdentifier(pos Pos) (string, MPos, error) {
	return mHandleMFn(pos, p.mIdentifier, func(mp MPos) (string, error) {
		return p.SourceStr(mp), nil
	})
}
func (p *Parser) mvQuotedString1(pos Pos) (string, MPos, error) {
	return mHandleMFn(pos, p.mQuotedString1, func(mp MPos) (string, error) {
		return strconv.Unquote(p.SourceStr(mp))
	})
}

//----------

func (p *Parser) mvTime(pos Pos, fmt string) (time.Time, MPos, error) {
	//return MHandleMFn(pos, p.WTime(fmt), func(mp MPos) (time.Time, error) {
	//	return time.Parse(fmt, p.SourceStr(mp))
	//})

	if s, mp, err := p.mvSourceStr(pos, p.maxNRunes(len(fmt))); err != nil {
		return time.Time{}, mp, err
	} else {
		t, err := time.Parse(fmt, s)
		return t, mp, err
	}
}

//----------

func mvOr[T any](pos Pos, fns ...VFn[T]) (T, MPos, error) {
	for _, fn := range fns {
		if v, mp, err := fn(pos); err != nil {
			if IsFatalError(err) {
				return v, mp, err
			}
		} else {
			return v, mp, nil
		}
	}
	var zero T
	return zero, MPos{pos, pos}, NoMatchErr
}

// Cast the value of fn to "T". Ex: useful to append to []T.
func mvCast[T, U any](pos Pos, fn VFn[U]) (T, MPos, error) {
	v, mp, err := fn(pos)
	if err != nil {
		var zero T
		return zero, mp, err
	}
	return any(v).(T), mp, err
}

// Cast the value of fn to "any". Ex: useful to append to []any.
func mvAny[T any](pos Pos, fn VFn[T]) (any, MPos, error) {
	return mvCast[any, T](pos, fn)
}

func mvConst[T any](pos Pos, fn MFn, v T) (T, MPos, error) {
	if mp, err := fn(pos); err != nil {
		var zero T
		return zero, mp, err
	} else {
		return v, mp, nil
	}
}

// ex: useful in the case of MVTime (doesn't have a MTime)
func mvToken[T any](p *Parser, pos Pos, fn VFn[T]) (T, MPos, error) {
	var v T
	mp, err := p.mToken(pos, keep(&v, fn))
	return v, mp, err
}

//----------

func mAssign[T any](pos Pos, v *T, fn VFn[T]) (MPos, error) {
	return mHandleVFn(pos, fn, func(v2 T) error {
		*v = v2
		return nil
	})
}
func mAssign2[T any](pos Pos, v **T, fn VFn[T]) (MPos, error) {
	return mHandleVFn(pos, fn, func(v2 T) error {
		*v = new(T)
		**v = v2
		return nil
	})
}

func mAppend[T any](pos Pos, w *[]T, fn VFn[T]) (MPos, error) {
	return mHandleVFn(pos, fn, func(v T) error {
		*w = append(*w, v)
		return nil
	})
}

func mSetMapEntry[K comparable, V any](pos Pos, m *map[K]V, fn VFn[MapEntry[K, V]]) (MPos, error) {
	return mHandleVFn(pos, fn, func(v MapEntry[K, V]) error {
		(*m)[v.Key] = v.Value
		return nil
	})
}

func mvAppend[T any](pos Pos, fn VFn[T]) ([]T, MPos, error) {
	w := []T{}
	mp, err := mAppend(pos, &w, fn)
	return w, mp, err
}

//func MZero[T any](pos Pos, v *T) (MPos, error) {
//	*v = *new(T)
//	return MPos{pos, pos}, nil
//}

//----------
//----------
//----------

func (p *Parser) mDebugAnd(pos Pos, on bool, prefix string, fns ...MFn) (MPos, error) {
	return p.mDebug(pos, on, prefix, p.and(fns...))
}
func (p *Parser) mDebugOr(pos Pos, on bool, prefix string, fns ...MFn) (MPos, error) {
	return p.mDebug(pos, on, prefix, p.or(fns...))
}
func (p *Parser) mDebug(pos Pos, on bool, prefix string, fn MFn) (MPos, error) {
	if !on {
		return fn(pos)
	}
	mp, err := fn(pos)
	if err != nil {
		b := BytesSnippet(p.src, mp, 20)
		fmt.Printf("[%s]:err: %s; %q\n", prefix, err, b)
	} else {
		b := p.Source(mp)
		fmt.Printf("[%s]:ok: %q\n", prefix, b)
	}
	return mp, err
}

func (p *Parser) mFatalOnError(pos Pos, tag string, fn MFn) (MPos, error) {
	if mp, err := fn(pos); err != nil {
		return mp, FatalError2(tag, err)
	} else {
		return mp, nil
	}
}

//----------

func (p *Parser) loopNoProgressError(tag string, pos Pos) error {
	return FatalError2(tag, fmt.Errorf("loop with no progress: %v", p.Snippet(MPos{pos, pos})))
}
