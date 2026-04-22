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

func mvByte(ps *ParserState, pos Pos) (byte, MPos, error) {
	if pos > ps.farthest {
		ps.farthest = pos
	}

	if pos < ps.srcMin || pos >= ps.srcMax {
		return 0, MPos{pos, pos}, NoMatchErr
	}
	b := ps.src[pos]
	p2 := pos + 1
	return b, MPos{pos, p2}, nil
}
func mvRune(ps *ParserState, pos Pos) (rune, MPos, error) {
	if pos > ps.farthest {
		ps.farthest = pos
	}

	if pos < ps.srcMin || pos >= ps.srcMax {
		return 0, MPos{pos, pos}, NoMatchErr
	}
	ru, size := utf8.DecodeRune(ps.src[pos:ps.srcMax])
	if size == 0 {
		return 0, MPos{pos, pos}, NoMatchErr
	}
	p2 := pos + Pos(size)
	return ru, MPos{pos, p2}, nil
}
func mvLastRune(ps *ParserState, pos Pos) (rune, MPos, error) {
	if pos <= ps.srcMin || pos > ps.srcMax {
		return 0, MPos{pos, pos}, NoMatchErr
	}
	ru, size := utf8.DecodeLastRune(ps.src[ps.srcMin:pos])
	if size == 0 {
		return 0, MPos{pos, pos}, NoMatchErr
	}
	p2 := pos - Pos(size)
	// Last rune moves backwards, so MPos intentionally has Start > End.
	return ru, MPos{pos, p2}, nil
}

//----------

// there can be no nested tokens; should be set at leaf nodes
func mToken(ps *ParserState, pos Pos, fn MFn) (MPos, error) {
	pos = runIgnore(ps, pos)

	ps.tokDepth++
	defer func() { ps.tokDepth-- }()
	if ps.tokDepth > 1 {
		err := fmt.Errorf("nested tokens: %v", ps.Snippet(MPos{pos, pos}))
		panic(err)
	}

	return fn(ps, pos)
}

func runIgnore(ps *ParserState, pos Pos) Pos {
	if ps.tokDepth > 0 {
		return pos
	}

	ps.ignore.depth++
	defer func() { ps.ignore.depth-- }()
	if ps.ignore.depth > 1 {
		return pos
	}

	if ps.Ignore != nil {
		if ps.ignore.cache.valid && ps.ignore.cache.pos == pos {
			return ps.ignore.cache.result
		}
		if mp, err := ps.Ignore(ps, pos); err == nil {
			ps.ignore.cache.valid = true
			ps.ignore.cache.pos = pos
			ps.ignore.cache.result = mp.End
			pos = mp.End
		}
	}

	return pos
}

//----------

func mAnd(ps *ParserState, pos Pos, fns ...MFn) (MPos, error) {
	p0 := pos
	for _, fn := range fns {
		if mp, err := fn(ps, pos); err != nil {
			return mp, err
		} else {
			pos = mp.End
		}
	}
	return MPos{p0, pos}, nil
}
func mOr(ps *ParserState, pos Pos, fns ...MFn) (MPos, error) {
	for _, fn := range fns {
		if mp, err := fn(ps, pos); err != nil {
			if IsFatalError(err) {
				return mp, err
			}
		} else {
			return mp, nil
		}
	}
	return MPos{pos, pos}, NoMatchErr
}

func mOptional(ps *ParserState, pos Pos, fn MFn) (MPos, error) {
	if mp, err := fn(ps, pos); err != nil {
		if IsFatalError(err) {
			return mp, err
		}
		return MPos{pos, pos}, nil
	} else {
		return mp, nil
	}
}
func mPeek(ps *ParserState, pos Pos, fn MFn) (MPos, error) {
	if mp, err := fn(ps, pos); err != nil {
		return mp, err
	} else {
		return MPos{pos, pos}, nil // stay in same pos
	}
}
func mWithBounds(ps *ParserState, pos Pos, back, forward int, fn MFn) (MPos, error) {
	start := ps.srcMin
	if back >= 0 {
		start = max(0, pos-Pos(back))
	}
	end := ps.srcMax
	if forward >= 0 {
		end = min(pos+Pos(forward), ps.sourceLen())
	}
	return mWithBoundsAbs(ps, pos, start, end, fn)
}

func mWithLineBounds(ps *ParserState, pos Pos, back, forward int, fn MFn) (MPos, error) {
	start := lineBoundsStart(ps.src, ps.srcMin, pos, back)
	end := lineBoundsEnd(ps.src, pos, ps.srcMax, forward)
	return mWithBoundsAbs(ps, pos, start, end, fn)
}

func mWithBoundsAbs(ps *ParserState, pos, start, end Pos, fn MFn) (MPos, error) {
	srcMin := ps.srcMin
	srcMax := ps.srcMax
	ps.srcMin = max(srcMin, start)
	ps.srcMax = min(srcMax, end)
	defer func() {
		ps.srcMin = srcMin
		ps.srcMax = srcMax
	}()
	return fn(ps, pos)
}

func mReverseSource(ps *ParserState, pos Pos, fn MFn) (MPos, error) {
	rev := reverseSourceBytes(ps.src[ps.srcMin:ps.srcMax])
	parsePos := ps.srcMax - pos

	ps2 := NewParserStateFromBytes(rev)

	ps2.UserData = ps.UserData
	ps2.parseStart = parsePos
	ps2.tokDepth = ps.tokDepth
	ps2.Ignore = ps.Ignore

	mp, err := fn(ps2, parsePos)
	if err != nil {
		return reverseSourceMPos(ps.srcMin, ps.srcMax, mp), err
	}
	return reverseSourceMPos(ps.srcMin, ps.srcMax, mp), nil
}
func mNot(ps *ParserState, pos Pos, fn MFn) (MPos, error) {
	if mp, err := fn(ps, pos); err != nil {
		return MPos{pos, pos}, nil
	} else {
		return mp, NoMatchErr
	}
}
func mFail(ps *ParserState, pos Pos) (MPos, error) {
	return MPos{pos, pos}, NoMatchErr
}
func mNoOp(ps *ParserState, pos Pos) (MPos, error) {
	return MPos{pos, pos}, nil
}
func mEof(ps *ParserState, pos Pos) (MPos, error) {
	if _, mp, err := mvByte(ps, pos); err != nil { // eof
		return MPos{pos, pos}, nil
	} else {
		return mp, NoMatchErr
	}
}

// start-of-file
func mSof(ps *ParserState, pos Pos) (MPos, error) {
	if pos == 0 {
		return MPos{pos, pos}, nil
	}
	return MPos{pos, pos}, NoMatchErr
}

func mByteFn(ps *ParserState, pos Pos, fn func(byte) bool) (MPos, error) {
	return mHandleVFn(ps, pos, mvByte, BoolErrFn(fn))
}
func mByteFnLoop(ps *ParserState, pos Pos, fn func(byte) bool) (MPos, error) {
	return mLoop1(ps, pos, byteFn(fn))
}

//----------

func mRuneFn(ps *ParserState, pos Pos, fn func(rune) bool) (MPos, error) {
	return mHandleVFn(ps, pos, mvRune, BoolErrFn(fn))
}
func mRuneFnLoop(ps *ParserState, pos Pos, fn func(rune) bool) (MPos, error) {
	return mLoop1(ps, pos, runeFn(fn))
}
func mRune(ps *ParserState, pos Pos, ru rune) (MPos, error) {
	return mRuneFn(ps, pos, func(ru2 rune) bool { return ru2 == ru })
}
func mContainsRune(ps *ParserState, pos Pos, s string) (MPos, error) {
	return mRuneFn(ps, pos, func(ru rune) bool {
		return strings.ContainsRune(s, ru)
	})
}
func mAnyRune(ps *ParserState, pos Pos) (MPos, error) {
	_, mp, err := mvRune(ps, pos)
	return mp, err
}
func mNRunes(ps *ParserState, pos Pos, n int) (MPos, error) {
	p0 := pos
	for k := 0; k < n; k++ {
		if _, mp, err := mvRune(ps, pos); err != nil {
			return MPos{p0, pos}, err
		} else {
			pos = mp.End
		}
	}
	return MPos{p0, pos}, nil
}
func mMaxNRunes(ps *ParserState, pos Pos, n int) (MPos, error) {
	p0 := pos
	for k := 0; k < n; k++ {
		if _, mp, err := mvRune(ps, pos); err != nil {
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
func mSeq(ps *ParserState, pos Pos, s string) (MPos, error) {
	if s == "" {
		return mFail(ps, pos)
	}
	p0 := pos
	for _, ru := range s {
		if ru2, mp, err := mvRune(ps, pos); err != nil {
			return MPos{p0, pos}, err
		} else if ru2 != ru {
			return MPos{p0, pos}, NoMatchErr
		} else {
			pos = mp.End
		}
	}
	return MPos{p0, pos}, nil
}

func mSeqOrMid(ps *ParserState, pos Pos, s string) (MPos, error) {
	n := Pos(len(s))
	for p0 := pos; ; {
		if p0+n <= pos {
			break
		}
		if mp, err := mSeq(ps, p0, s); err == nil {
			return mp, nil
		}

		if p0 == 0 {
			break
		}
		_, mp, err := mvLastRune(ps, p0)
		if err != nil {
			break
		}
		p0 = mp.End
	}
	return MPos{pos, pos}, NoMatchErr
}

//----------

func lineBoundsStart(src []byte, min, pos Pos, back int) Pos {
	i := pos
	n := 0
	for i > min {
		if src[i-1] == '\n' {
			if n >= back {
				break
			}
			n++
		}
		i--
	}
	return i
}

func lineBoundsEnd(src []byte, pos, max Pos, forward int) Pos {
	i := pos
	n := 0
	for i < max {
		if src[i] == '\n' {
			if n >= forward {
				break
			}
			n++
		}
		i++
	}
	return i
}

func reverseSourceBytes(src []byte) []byte {
	rs := []rune(string(src))
	slices.Reverse(rs)
	return []byte(string(rs))
}

func reverseSourceMPos(srcMin, srcMax Pos, mp MPos) MPos {
	return MPos{
		Start: srcMin + (srcMax - srcMin - mp.Start),
		End:   srcMin + (srcMax - srcMin - mp.End),
	}
}

//----------

func mLoop0(ps *ParserState, pos Pos, maxN int, fn MFn) (MPos, int, error) {
	p0 := pos
	for k := 0; maxN < 0 || k < maxN; k++ {
		mp, err := fn(ps, pos)
		if err != nil {
			if IsFatalError(err) {
				return mp, k, err
			}
			return MPos{p0, pos}, k, err
		}
		if mp.End == pos {
			return mp, k, loopNoProgressError(ps, "mLoop0", pos)
		}
		pos = mp.End
	}
	return MPos{p0, pos}, maxN, nil
}

func mLoop1(ps *ParserState, pos Pos, fn MFn) (MPos, error) {
	mp, n, err := mLoop0(ps, pos, -1, fn)
	if err != nil {
		if n == 0 {
			return mp, err
		}
		return MPos{pos, mp.End}, nil
	}
	return mp, nil
}

func mLoopSep(ps *ParserState, pos Pos, optLastSep, allowEmpty bool, fn, sepFn MFn) (MPos, error) {
	p0 := pos
	for k := 0; ; k++ {
		// separator
		p2 := pos
		if k > 0 {
			if mp, err := sepFn(ps, pos); err != nil {
				return MPos{p0, pos}, nil
			} else {
				if mp.End == pos {
					return mp, loopNoProgressError(ps, "mLoopSesep", pos)
				}
				pos = mp.End
			}
		}

		if mp, err := fn(ps, pos); err != nil {
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
				if !allowEmpty {
					return mp, loopNoProgressError(ps, "mLoopSefn", pos)
				}
			}
			pos = mp.End
		}
	}
}
func mLoopConsumeUntil(ps *ParserState, pos Pos, consumeFn, untilFn MFn) (MPos, error) {
	p0 := pos
	for {
		if mp, err := untilFn(ps, pos); err == nil {
			return MPos{p0, mp.End}, nil
		} else if IsFatalError(err) {
			return mp, err
		}

		if mp, err := consumeFn(ps, pos); err != nil {
			if IsFatalError(err) {
				return mp, err
			}
			return untilFn(ps, pos)
		} else {
			if mp.End == pos {
				return mp, loopNoProgressError(ps, "mLoopConsumeUntil", pos)
			}
			pos = mp.End
		}
	}
}

func mLoopToNLOrEof(ps *ParserState, pos Pos, esc rune, includeNL bool) (MPos, error) {
	nlFn := (MFn)(nil)
	if includeNL {
		nlFn = mNewline
	} else {
		nlFn = peek(mNewline)
	}
	return mLoopConsumeUntil(ps, pos,
		or(
			escape(esc),
			mAnyRune,
		),
		or(
			nlFn,
			mEof,
		),
	)
}

//----------

func mDigit(ps *ParserState, pos Pos) (MPos, error) {
	return mRuneFn(ps, pos, unicode.IsDigit)
}
func mDigitNotZero(ps *ParserState, pos Pos) (MPos, error) {
	return mRuneFn(ps, pos, func(ru rune) bool {
		return ru != '0' && unicode.IsDigit(ru)
	})
}
func mDigits(ps *ParserState, pos Pos) (MPos, error) {
	return mLoop1(ps, pos, mDigit)
}

func mFloat2(ps *ParserState, pos Pos, sep rune) (MPos, error) {
	return mAnd(ps, pos,
		//p.WOptional(p.MInteger), // wrong, won't allow "-0.1"

		optional(mSign),
		or(
			and(
				rune1('0'),
				// avoid 2nd zero
				optional(and(
					mDigitNotZero,
					optional(mDigits),
				)),
			),
			and(
				mDigitNotZero,
				optional(mDigits),
			),
		),

		// fraction
		and(
			rune1(sep),
			mDigits,
		),

		// TODO: exponent?
	)
}
func mInteger(ps *ParserState, pos Pos) (MPos, error) {
	return mOr(ps, pos,
		and(
			optional(mSign),
			mDigitNotZero,
			optional(mDigits),
		),
		// just zero
		and(
			rune1('0'),
			peek(not(mDigits)),
		),
	)
}
func mSign(ps *ParserState, pos Pos) (MPos, error) {
	return mContainsRune(ps, pos, "-+")
}
func mBool(ps *ParserState, pos Pos) (MPos, error) {
	return mOr(ps, pos,
		seq("true"), seq("false"),
		seq("True"), seq("False"),
		seq("TRUE"), seq("FALSE"),
	)
}

// NOTE: use p.MVTime
//// TODO: this a simple/fixed/rigid time, needs improvement
//// TODO: ex: fmt "_2" won't match "2"
//func MTime(ps *ParserState, pos Pos, fmt string) (MPos, error) {
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

func mNewline(ps *ParserState, pos Pos) (MPos, error) {
	return mRune(ps, pos, '\n')
}

//----------

// the ignore fn should not consume newlines
func mEmptyLinesExceptNewline(ps *ParserState, pos Pos, ignore MFn) (MPos, error) {
	ignores := loop1(ignore)
	return mOr(ps, pos,
		// start of file (special case): also consumes ending newlines
		and(
			mSof,
			loop1(or(
				ignores,
				mNewline,
			)),
		),
		// middle of file
		loop1(or(
			ignores,
			// empty lines
			and(
				mNewline,
				optional(ignores),
				peek(mNewline),
			),
		)),
	)
}

//----------

func mIdentifier(ps *ParserState, pos Pos) (MPos, error) {
	return mAnd(ps, pos,
		runeFn(func(ru rune) bool {
			return unicode.IsLetter(ru) || ru == '_'
		}),
		optional(runeFnLoop(func(ru rune) bool {
			return unicode.IsLetter(ru) || unicode.IsDigit(ru) || ru == '_'
		})),
	)
}
func mEscape(ps *ParserState, pos Pos, esc rune) (MPos, error) {
	if esc == 0 {
		return mFail(ps, pos)
	}
	return mAnd(ps, pos,
		rune1(esc),
		nRunes(1),
	)
}
func mAnyExceptNewline(ps *ParserState, pos Pos) (MPos, error) {
	return mAnd(ps, pos,
		not(rune1('\n')),
		mAnyRune,
	)
}

//----------

func mQuotedString1(ps *ParserState, pos Pos) (MPos, error) {
	return mSection(ps, pos, "\"", "\"", '\\', false, false, mAnyExceptNewline)
}
func mSection(ps *ParserState, pos Pos,
	open, close string,
	esc rune,
	newlineCloses, eofCloses bool,
	consume MFn) (MPos, error) {

	closeFn := seq(close)
	if eofCloses {
		closeFn = or(
			mEof,
			closeFn,
		)
	}
	if newlineCloses {
		closeFn = or(
			peek(rune1('\n')), // don't consume
			closeFn,
		)
	}
	return mAnd(ps, pos,
		seq(open),
		loopConsumeUntil(
			or(
				escape(esc),
				consume,
			),
			closeFn,
		),
	)
}

//----------
//----------
//----------

func mHandleMFn[T any](ps *ParserState, pos Pos, fn1 MFn, fn2 VMaker[T]) (T, MPos, error) {
	if mp, err := fn1(ps, pos); err != nil {
		var zero T
		return zero, mp, err
	} else {
		v, err := fn2(mp)
		return v, mp, err
	}
}
func mHandleVFn[T any](ps *ParserState, pos Pos, fn1 VFn[T], fn2 VHandler[T]) (MPos, error) {
	if v, mp, err := fn1(ps, pos); err != nil {
		return mp, err
	} else {
		return mp, fn2(v)
	}
}

//----------

func mvBytes(ps *ParserState, pos Pos, fn MFn) ([]byte, MPos, error) {
	return mHandleMFn(ps, pos, fn, func(mp MPos) ([]byte, error) {
		return ps.Source(mp), nil
	})
}
func mvString(ps *ParserState, pos Pos, fn MFn) (string, MPos, error) {
	b, mp, err := mvBytes(ps, pos, fn)
	return string(b), mp, err
}

func mvFloat(ps *ParserState, pos Pos) (float64, MPos, error) {
	return mHandleMFn(ps, pos, func(ps *ParserState, pos Pos) (MPos, error) {
		return mFloat2(ps, pos, '.')
	}, func(mp MPos) (float64, error) {
		return strconv.ParseFloat(ps.SourceStr(mp), 64)
	})
}
func mvInteger(ps *ParserState, pos Pos) (int, MPos, error) {
	return mHandleMFn(ps, pos, mInteger, func(mp MPos) (int, error) {
		v, err := strconv.ParseInt(ps.SourceStr(mp), 10, 64)
		return int(v), err
	})
}
func mvBool(ps *ParserState, pos Pos) (bool, MPos, error) {
	return mHandleMFn(ps, pos, mBool, func(mp MPos) (bool, error) {
		return strconv.ParseBool(ps.SourceStr(mp))
	})
}
func mvIdentifier(ps *ParserState, pos Pos) (string, MPos, error) {
	return mHandleMFn(ps, pos, mIdentifier, func(mp MPos) (string, error) {
		return ps.SourceStr(mp), nil
	})
}
func mvQuotedString1(ps *ParserState, pos Pos) (string, MPos, error) {
	return mHandleMFn(ps, pos, mQuotedString1, func(mp MPos) (string, error) {
		return strconv.Unquote(ps.SourceStr(mp))
	})
}

//----------

func mvTime(ps *ParserState, pos Pos, fmt string) (time.Time, MPos, error) {
	//return MHandleMFn(pos, p.WTime(fmt), func(mp MPos) (time.Time, error) {
	//	return time.Parse(fmt, p.SourceStr(mp))
	//})

	if s, mp, err := mvString(ps, pos, maxNRunes(len(fmt))); err != nil {
		return time.Time{}, mp, err
	} else {
		t, err := time.Parse(fmt, s)
		return t, mp, err
	}
}

//----------

func mvOr[T any](ps *ParserState, pos Pos, fns ...VFn[T]) (T, MPos, error) {
	for _, fn := range fns {
		if v, mp, err := fn(ps, pos); err != nil {
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
func mvCast[T, U any](ps *ParserState, pos Pos, fn VFn[U]) (T, MPos, error) {
	v, mp, err := fn(ps, pos)
	if err != nil {
		var zero T
		return zero, mp, err
	}
	return any(v).(T), mp, err
}

// Cast the value of fn to "any". Ex: useful to append to []any.
func mvAny[T any](ps *ParserState, pos Pos, fn VFn[T]) (any, MPos, error) {
	return mvCast[any, T](ps, pos, fn)
}

func mvConst[T any](ps *ParserState, pos Pos, fn MFn, v T) (T, MPos, error) {
	if mp, err := fn(ps, pos); err != nil {
		var zero T
		return zero, mp, err
	} else {
		return v, mp, nil
	}
}

// ex: useful in the case of MVTime (doesn't have a MTime)
func mvToken[T any](ps *ParserState, pos Pos, fn VFn[T]) (T, MPos, error) {
	var v T
	mp, err := mToken(ps, pos, assign(&v, fn))
	return v, mp, err
}

//----------

func mAssign[T any](ps *ParserState, pos Pos, v *T, fn VFn[T]) (MPos, error) {
	return mHandleVFn(ps, pos, fn, func(v2 T) error {
		*v = v2
		return nil
	})
}
func mAssign2[T any](ps *ParserState, pos Pos, v **T, fn VFn[T]) (MPos, error) {
	return mHandleVFn(ps, pos, fn, func(v2 T) error {
		*v = new(T)
		**v = v2
		return nil
	})
}

func mAppend[T any](ps *ParserState, pos Pos, w *[]T, fn VFn[T]) (MPos, error) {
	return mHandleVFn(ps, pos, fn, func(v T) error {
		*w = append(*w, v)
		return nil
	})
}

func mSetMapEntry[K comparable, V any](ps *ParserState, pos Pos, m *map[K]V, fn VFn[MapEntry[K, V]]) (MPos, error) {
	return mHandleVFn(ps, pos, fn, func(v MapEntry[K, V]) error {
		(*m)[v.Key] = v.Value
		return nil
	})
}

func mvAppend[T any](ps *ParserState, pos Pos, fn VFn[T]) ([]T, MPos, error) {
	w := []T{}
	mp, err := mAppend(ps, pos, &w, fn)
	return w, mp, err
}

//func MZero[T any](pos Pos, v *T) (MPos, error) {
//	*v = *new(T)
//	return MPos{pos, pos}, nil
//}

//----------
//----------
//----------

func and(fns ...MFn) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mAnd(ps, pos, fns...)
	}
}

func or(fns ...MFn) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mOr(ps, pos, fns...)
	}
}

func optional(fn MFn) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mOptional(ps, pos, fn)
	}
}

func peek(fn MFn) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mPeek(ps, pos, fn)
	}
}

func not(fn MFn) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mNot(ps, pos, fn)
	}
}

func byteFn(fn func(byte) bool) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mByteFn(ps, pos, fn)
	}
}

func runeFn(fn func(rune) bool) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mRuneFn(ps, pos, fn)
	}
}

func runeFnLoop(fn func(rune) bool) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mRuneFnLoop(ps, pos, fn)
	}
}

func rune1(ru rune) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mRune(ps, pos, ru)
	}
}

func nRunes(n int) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mNRunes(ps, pos, n)
	}
}

func maxNRunes(n int) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mMaxNRunes(ps, pos, n)
	}
}

func seq(s string) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mSeq(ps, pos, s)
	}
}

func loop1(fn MFn) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mLoop1(ps, pos, fn)
	}
}

func loopConsumeUntil(consumeFn, untilFn MFn) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mLoopConsumeUntil(ps, pos, consumeFn, untilFn)
	}
}

func escape(esc rune) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mEscape(ps, pos, esc)
	}
}

func assign[T any](v *T, fn VFn[T]) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mAssign(ps, pos, v, fn)
	}
}

func mDebugAnd(ps *ParserState, pos Pos, on bool, prefix string, fns ...MFn) (MPos, error) {
	return mDebug(ps, pos, on, prefix, and(fns...))
}
func mDebugOr(ps *ParserState, pos Pos, on bool, prefix string, fns ...MFn) (MPos, error) {
	return mDebug(ps, pos, on, prefix, or(fns...))
}
func mDebug(ps *ParserState, pos Pos, on bool, prefix string, fn MFn) (MPos, error) {
	if !on {
		return fn(ps, pos)
	}
	mp, err := fn(ps, pos)
	if err != nil {
		b := BytesSnippet(ps.src, mp, 20)
		fmt.Printf("[%s]:err: %s; %q\n", prefix, err, b)
	} else {
		b := ps.Source(mp)
		fmt.Printf("[%s]:ok: %q\n", prefix, b)
	}
	return mp, err
}

func mFatalOnError(ps *ParserState, pos Pos, tag string, fn MFn) (MPos, error) {
	if mp, err := fn(ps, pos); err != nil {
		return mp, FatalError2(tag, err)
	} else {
		return mp, nil
	}
}

//----------

func loopNoProgressError(ps *ParserState, tag string, pos Pos) error {
	return FatalError2(tag, fmt.Errorf("loop with no progress: %v", ps.Snippet(MPos{pos, pos})))
}
