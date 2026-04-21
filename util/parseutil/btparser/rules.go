package btparser

import (
	"errors"
	"fmt"
	"time"
)

// Rules is the main public API for building grammars.
// Rules owns the grammar and combinators, while ParserState owns the source and runtime parse state for a single parse.
// Rules instances are intended for sequential use; Parse mutates no ParserState-independent execution state, but SetIgnore changes parser configuration.
type Rules struct {
	ignore MFn
}

func NewRules() Rules {
	return Rules{}
}

func (g *Rules) SetIgnore(fn MFn) {
	g.ignore = fn
}

func (g Rules) Parse(ps *ParserState, fn MFn) (Pos, error) {
	return g.ParseAt(ps, 0, fn)
}

func (g Rules) ParseAt(ps *ParserState, pos Pos, fn MFn) (Pos, error) {
	ps.parseStart = pos
	mp, err := fn(ps, pos)
	if err != nil {
		isFatal := IsFatalError(err)

		err = fmt.Errorf("%v: %q", err, ps.Snippet(mp))

		mp2 := MPos{ps.farthest, ps.farthest}
		if mp != mp2 {
			err2 := fmt.Errorf("farthest: %q", ps.Snippet(mp2))
			err = errors.Join(err, err2)
		}
		if isFatal {
			err = FatalError(err)
		}

		return mp.End, err
	}
	return mp.End, nil
}

func (g Rules) VByte() VFn[byte]          { return vByte() }
func (g Rules) VRune() VFn[rune]          { return vRune() }
func (g Rules) VLastRune() VFn[rune]      { return vLastRune() }
func (g Rules) Token(fn MFn) MFn          { return token(g.ignore, fn) }
func (g Rules) And(fns ...MFn) MFn        { return and(fns...) }
func (g Rules) ReverseAnd(fns ...MFn) MFn { return reverseAnd(fns...) }
func (g Rules) Or(fns ...MFn) MFn         { return or(fns...) }
func (g Rules) Optional(fn MFn) MFn       { return optional(fn) }
func (g Rules) Peek(fn MFn) MFn           { return peek(fn) }
func (g Rules) LimitSourceBytes(back, forward int, fn MFn) MFn {
	return limitSourceBytes(back, forward, fn)
}
func (g Rules) LimitSourceLines(back, forward int, fn MFn) MFn {
	return limitSourceLines(back, forward, fn)
}
func (g Rules) ReverseSource(fn MFn) MFn {
	return reverseSource(fn)
}
func (g Rules) PeekBackN(n int, fn MFn) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mPeekBackN(ps, pos, n, fn)
	}
}
func (g Rules) Not(fn MFn) MFn { return not(fn) }
func (g Rules) Fail() MFn      { return mFail }
func (g Rules) NoOp() MFn      { return mNoOp }
func (g Rules) IsTrue(v bool) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mIsTrue(ps, pos, v)
	}
}
func (g Rules) NoOp2(fn func() error) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mNoOp2(ps, pos, fn)
	}
}
func (g Rules) Eof() MFn { return mEof }

// Sof matches only at the start of the source.
func (g Rules) Sof() MFn { return mSof }

// Sop matches only at the start position of the current parse.
func (g Rules) Sop() MFn { return mSop }

func (g Rules) ByteFn(fn func(byte) bool) MFn { return byteFn(fn) }
func (g Rules) ByteFnLoop(fn func(byte) bool) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mByteFnLoop(ps, pos, fn)
	}
}
func (g Rules) Byte(b byte) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mByte(ps, pos, b)
	}
}
func (g Rules) RuneFn(fn func(rune) bool) MFn     { return runeFn(fn) }
func (g Rules) RuneFnLoop(fn func(rune) bool) MFn { return runeFnLoop(fn) }
func (g Rules) Rune(ru rune) MFn                  { return rune1(ru) }
func (g Rules) RuneAnyOf(rus ...rune) MFn         { return runeAnyOf(rus...) }
func (g Rules) RuneAnyOfString(s string) MFn {
	return runeAnyOf([]rune(s)...)
}
func (g Rules) AnyRune() MFn          { return mAnyRune }
func (g Rules) NRunes(n int) MFn      { return nRunes(n) }
func (g Rules) MaxNRunes(n int) MFn   { return maxNRunes(n) }
func (g Rules) Seq(s string) MFn      { return seq(s) }
func (g Rules) SeqOrMid(s string) MFn { return seqOrMid(s) }
func (g Rules) Loop1(fn MFn) MFn      { return loop1(fn) }
func (g Rules) Loop2(minN, maxN int, fn MFn) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mLoop2(ps, pos, minN, maxN, fn)
	}
}
func (g Rules) LoopSep(optLastSep bool, fn, sepFn MFn) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mLoopSep(ps, pos, optLastSep, fn, sepFn)
	}
}
func (g Rules) LoopStartEnd(startFn, consumeFn, endFn MFn) MFn {
	return loopStartEnd(startFn, consumeFn, endFn)
}
func (g Rules) LoopToNLOrEof(esc rune, includeNL bool) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mLoopToNLOrEof(ps, pos, esc, includeNL)
	}
}
func (g Rules) Letter() MFn       { return mLetter }
func (g Rules) Digit() MFn        { return mDigit }
func (g Rules) DigitNotZero() MFn { return mDigitNotZero }
func (g Rules) Digits() MFn       { return mDigits }
func (g Rules) Float() MFn        { return mFloat }
func (g Rules) Float2(sep rune) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mFloat2(ps, pos, sep)
	}
}
func (g Rules) Integer() MFn             { return mInteger }
func (g Rules) Bool() MFn                { return mBool }
func (g Rules) Sign() MFn                { return mSign }
func (g Rules) HexBytes() MFn            { return mHexBytes }
func (g Rules) Space() MFn               { return mSpace }
func (g Rules) Spaces() MFn              { return mSpaces }
func (g Rules) SpacesExceptNewline() MFn { return mSpacesExceptNewline }
func (g Rules) Newline() MFn             { return mNewline }
func (g Rules) EmptyLinesExceptNewline(ignore MFn) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mEmptyLinesExceptNewline(ps, pos, ignore)
	}
}
func (g Rules) Escape(esc rune) MFn   { return escape(esc) }
func (g Rules) AnyExceptNewline() MFn { return mAnyExceptNewline }
func (g Rules) Identifier() MFn       { return mIdentifier }
func (g Rules) QuotedString1() MFn    { return mQuotedString1 }
func (g Rules) LineComment1(open string) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mLineComment1(ps, pos, open)
	}
}
func (g Rules) Section(open, close string, esc rune, newlineCloses, eofCloses bool, consume MFn) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mSection(ps, pos, open, close, esc, newlineCloses, eofCloses, consume)
	}
}
func (g Rules) VSource(fn MFn) VFn[[]byte] {
	return func(ps *ParserState, pos Pos) ([]byte, MPos, error) {
		return mvSource(ps, pos, fn)
	}
}
func (g Rules) VSourceStr(fn MFn) VFn[string] { return sourceStr(fn) }
func (g Rules) VFloat() VFn[float64]          { return mvFloat }
func (g Rules) VInteger() VFn[int]            { return mvInteger }
func (g Rules) VBool() VFn[bool]              { return mvBool }
func (g Rules) VIdentifier() VFn[string]      { return mvIdentifier }
func (g Rules) VQuotedString1() VFn[string]   { return mvQuotedString1 }
func (g Rules) VTime(fmt string) VFn[time.Time] {
	return func(ps *ParserState, pos Pos) (time.Time, MPos, error) {
		return mvTime(ps, pos, fmt)
	}
}
func (g Rules) DebugAnd(on bool, prefix string, fns ...MFn) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mDebugAnd(ps, pos, on, prefix, fns...)
	}
}
func (g Rules) DebugOr(on bool, prefix string, fns ...MFn) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mDebugOr(ps, pos, on, prefix, fns...)
	}
}
func (g Rules) Debug(on bool, prefix string, fn MFn) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mDebug(ps, pos, on, prefix, fn)
	}
}
func (g Rules) FatalOnError(tag string, fn MFn) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mFatalOnError(ps, pos, tag, fn)
	}
}

//----------
//----------
//----------

type MapEntry[K comparable, V any] struct {
	Key   K
	Value V
}

//----------
//----------
//----------

func VOr[T any](fns ...VFn[T]) VFn[T] {
	return func(ps *ParserState, pos Pos) (T, MPos, error) {
		return mvOr(ps, pos, fns...)
	}
}

func VCast[T, U any](fn VFn[U]) VFn[T] {
	return func(ps *ParserState, pos Pos) (T, MPos, error) {
		return mvCast[T, U](ps, pos, fn)
	}
}

func VAny[T any](fn VFn[T]) VFn[any] {
	return func(ps *ParserState, pos Pos) (any, MPos, error) {
		return mvAny(ps, pos, fn)
	}
}

func VConst[T any](fn MFn, v T) VFn[T] {
	return func(ps *ParserState, pos Pos) (T, MPos, error) {
		return mvConst(ps, pos, fn, v)
	}
}

func VToken[T any](g Rules, fn VFn[T]) VFn[T] {
	return func(ps *ParserState, pos Pos) (T, MPos, error) {
		return mvToken(g, ps, pos, fn)
	}
}

func Assign[T any](v *T, fn VFn[T]) MFn {
	return assign(v, fn)
}

func Assign2[T any](v **T, fn VFn[T]) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mAssign2(ps, pos, v, fn)
	}
}

func Append[T any](w *[]T, fn VFn[T]) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mAppend(ps, pos, w, fn)
	}
}

func SetMapEntry[K comparable, V any](m *map[K]V, fn VFn[MapEntry[K, V]]) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mSetMapEntry(ps, pos, m, fn)
	}
}

func VAppend[T any](fn VFn[T]) VFn[[]T] {
	return func(ps *ParserState, pos Pos) ([]T, MPos, error) {
		return mvAppend(ps, pos, fn)
	}
}
