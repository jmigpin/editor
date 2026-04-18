package btparser

import "time"

// Rules is the main public API for building grammars.
// Typical usage:
//
//	p := NewParser()
//	g := p.G()
//	fn := g.And(
//		g.Token(g.Seq("abc")),
//		g.Token(g.EOF()),
//	)
type Rules struct {
	p *Parser
}

func NewRules(p *Parser) Rules {
	return Rules{p: p}
}

func (g Rules) VByte() VFn[byte]     { return g.p.vByte() }
func (g Rules) VRune() VFn[rune]     { return g.p.vRune() }
func (g Rules) VLastRune() VFn[rune] { return g.p.vLastRune() }
func (g Rules) Token(fn MFn) MFn     { return g.p.token(fn) }
func (g Rules) And(fns ...MFn) MFn   { return g.p.and(fns...) }
func (g Rules) Or(fns ...MFn) MFn    { return g.p.or(fns...) }
func (g Rules) Optional(fn MFn) MFn  { return g.p.optional(fn) }
func (g Rules) Opt(fn MFn) MFn       { return g.Optional(fn) }
func (g Rules) Lookahead(fn MFn) MFn { return g.p.lookahead(fn) }
func (g Rules) LookbackN(n int, fn MFn) MFn {
	return func(pos Pos) (MPos, error) { return g.p.mLookbackN(pos, n, fn) }
}
func (g Rules) Not(fn MFn) MFn { return g.p.not(fn) }
func (g Rules) Fail() MFn      { return g.p.mFail }
func (g Rules) NoOp() MFn      { return g.p.mNoOp }
func (g Rules) NoOp2(fn func() error) MFn {
	return func(pos Pos) (MPos, error) { return g.p.mNoOp2(pos, fn) }
}
func (g Rules) Eof() MFn                      { return g.p.mEof }
func (g Rules) EOF() MFn                      { return g.Eof() }
func (g Rules) Sof() MFn                      { return g.p.mSof }
func (g Rules) SOF() MFn                      { return g.Sof() }
func (g Rules) ByteFn(fn func(byte) bool) MFn { return g.p.byteFn(fn) }
func (g Rules) ByteFnLoop(fn func(byte) bool) MFn {
	return func(pos Pos) (MPos, error) { return g.p.mByteFnLoop(pos, fn) }
}
func (g Rules) Byte(b byte) MFn                   { return func(pos Pos) (MPos, error) { return g.p.mByte(pos, b) } }
func (g Rules) RuneFn(fn func(rune) bool) MFn     { return g.p.runeFn(fn) }
func (g Rules) RuneFnLoop(fn func(rune) bool) MFn { return g.p.runeFnLoop(fn) }
func (g Rules) Rune(ru rune) MFn                  { return g.p.rune(ru) }
func (g Rules) ContainsRune(s string) MFn {
	return func(pos Pos) (MPos, error) { return g.p.mContainsRune(pos, s) }
}
func (g Rules) ContainsRune2(rus []rune) MFn {
	return func(pos Pos) (MPos, error) { return g.p.mContainsRune2(pos, rus) }
}
func (g Rules) AnyRune() MFn        { return g.p.mAnyRune }
func (g Rules) NRunes(n int) MFn    { return g.p.nRunes(n) }
func (g Rules) MaxNRunes(n int) MFn { return g.p.maxNRunes(n) }
func (g Rules) Seq(s string) MFn    { return g.p.seq(s) }
func (g Rules) Loop1(fn MFn) MFn    { return g.p.loop1(fn) }
func (g Rules) Many(fn MFn) MFn     { return g.Loop1(fn) }
func (g Rules) Loop2(minN, maxN int, fn MFn) MFn {
	return func(pos Pos) (MPos, error) { return g.p.mLoop2(pos, minN, maxN, fn) }
}
func (g Rules) Repeat(minN, maxN int, fn MFn) MFn { return g.Loop2(minN, maxN, fn) }
func (g Rules) LoopSep(optLastSep bool, fn, sepFn MFn) MFn {
	return func(pos Pos) (MPos, error) { return g.p.mLoopSep(pos, optLastSep, fn, sepFn) }
}
func (g Rules) ManySep(optLastSep bool, fn, sepFn MFn) MFn { return g.LoopSep(optLastSep, fn, sepFn) }
func (g Rules) LoopStartEnd(startFn, consumeFn, endFn MFn) MFn {
	return g.p.loopStartEnd(startFn, consumeFn, endFn)
}
func (g Rules) LoopToNLOrEof(esc rune, includeNL bool) MFn {
	return func(pos Pos) (MPos, error) { return g.p.mLoopToNLOrEof(pos, esc, includeNL) }
}
func (g Rules) UntilNewlineOrEOF(esc rune, includeNL bool) MFn {
	return g.LoopToNLOrEof(esc, includeNL)
}
func (g Rules) Letter() MFn       { return g.p.mLetter }
func (g Rules) Digit() MFn        { return g.p.mDigit }
func (g Rules) DigitNotZero() MFn { return g.p.mDigitNotZero }
func (g Rules) Digits() MFn       { return g.p.mDigits }
func (g Rules) Float2(sep rune) MFn {
	return func(pos Pos) (MPos, error) { return g.p.mFloat2(pos, sep) }
}
func (g Rules) Integer() MFn             { return g.p.mInteger }
func (g Rules) Sign() MFn                { return g.p.mSign }
func (g Rules) HexBytes() MFn            { return g.p.mHexBytes }
func (g Rules) Space() MFn               { return g.p.mSpace }
func (g Rules) Spaces() MFn              { return g.p.mSpaces }
func (g Rules) SpacesExceptNewline() MFn { return g.p.mSpacesExceptNewline }
func (g Rules) Newline() MFn             { return g.p.mNewline }
func (g Rules) EmptyLinesExceptNewline(ignore MFn) MFn {
	return func(pos Pos) (MPos, error) { return g.p.mEmptyLinesExceptNewline(pos, ignore) }
}
func (g Rules) SkipEmptyLines(ignore MFn) MFn { return g.EmptyLinesExceptNewline(ignore) }
func (g Rules) Escape(esc rune) MFn           { return g.p.escape(esc) }
func (g Rules) AnyExceptNewline() MFn         { return g.p.mAnyExceptNewline }
func (g Rules) QuotedString1() MFn            { return g.p.mQuotedString1 }
func (g Rules) LineComment1(open string) MFn {
	return func(pos Pos) (MPos, error) { return g.p.mLineComment1(pos, open) }
}
func (g Rules) Section(open, close string, esc rune, newlineCloses, eofCloses bool, consume MFn) MFn {
	return func(pos Pos) (MPos, error) {
		return g.p.mSection(pos, open, close, esc, newlineCloses, eofCloses, consume)
	}
}
func (g Rules) VSource(fn MFn) VFn[[]byte] {
	return func(pos Pos) ([]byte, MPos, error) { return g.p.mvSource(pos, fn) }
}
func (g Rules) VSourceStr(fn MFn) VFn[string] { return g.p.sourceStr(fn) }
func (g Rules) VFloat() VFn[float64]          { return g.p.mvFloat }
func (g Rules) VInteger() VFn[int]            { return g.p.mvInteger }
func (g Rules) VBool() VFn[bool]              { return g.p.mvBool }
func (g Rules) VIdentifier() VFn[string]      { return g.p.mvIdentifier }
func (g Rules) VQuotedString1() VFn[string]   { return g.p.mvQuotedString1 }
func (g Rules) VTime(fmt string) VFn[time.Time] {
	return func(pos Pos) (time.Time, MPos, error) { return g.p.mvTime(pos, fmt) }
}
func (g Rules) DebugAnd(on bool, prefix string, fns ...MFn) MFn {
	return func(pos Pos) (MPos, error) { return g.p.mDebugAnd(pos, on, prefix, fns...) }
}
func (g Rules) DebugOr(on bool, prefix string, fns ...MFn) MFn {
	return func(pos Pos) (MPos, error) { return g.p.mDebugOr(pos, on, prefix, fns...) }
}
func (g Rules) Debug(on bool, prefix string, fn MFn) MFn {
	return func(pos Pos) (MPos, error) { return g.p.mDebug(pos, on, prefix, fn) }
}
func (g Rules) FatalOnError(tag string, fn MFn) MFn {
	return func(pos Pos) (MPos, error) { return g.p.mFatalOnError(pos, tag, fn) }
}

func (g Rules) BoolMatch() MFn       { return g.p.mBool }
func (g Rules) FloatMatch() MFn      { return g.p.mFloat }
func (g Rules) IntegerMatch() MFn    { return g.p.mInteger }
func (g Rules) IdentifierMatch() MFn { return g.p.mIdentifier }
func (g Rules) QuotedStringMatch() MFn {
	return g.p.mQuotedString1
}

func (g Rules) Int() VFn[int]                   { return g.VInteger() }
func (g Rules) Float() VFn[float64]             { return g.VFloat() }
func (g Rules) Bool() VFn[bool]                 { return g.VBool() }
func (g Rules) Identifier() VFn[string]         { return g.VIdentifier() }
func (g Rules) QuotedString() VFn[string]       { return g.VQuotedString1() }
func (g Rules) Time(fmt string) VFn[time.Time]  { return g.VTime(fmt) }
func (g Rules) Source(fn MFn) VFn[[]byte]       { return g.VSource(fn) }
func (g Rules) SourceString(fn MFn) VFn[string] { return g.VSourceStr(fn) }

//----------
//----------
//----------

func VOr[T any](fns ...VFn[T]) VFn[T] {
	return func(pos Pos) (T, MPos, error) {
		return mvOr(pos, fns...)
	}
}

func Choice[T any](fns ...VFn[T]) VFn[T] { return VOr(fns...) }

func VCast[T, U any](fn VFn[U]) VFn[T] {
	return func(pos Pos) (T, MPos, error) {
		return mvCast[T, U](pos, fn)
	}
}

func VAny[T any](fn VFn[T]) VFn[any] {
	return func(pos Pos) (any, MPos, error) {
		return mvAny(pos, fn)
	}
}

func Any(fn VFn[any]) VFn[any]        { return fn }
func AsAny[T any](fn VFn[T]) VFn[any] { return VAny(fn) }
func AnyOf(fns ...VFn[any]) VFn[any]  { return Choice(fns...) }

func VToken[T any](p *Parser, fn VFn[T]) VFn[T] {
	return func(pos Pos) (T, MPos, error) {
		return mvToken(p, pos, fn)
	}
}

func TokenValue[T any](p *Parser, fn VFn[T]) VFn[T] { return VToken(p, fn) }

func Keep[T any](v *T, fn VFn[T]) MFn {
	return keep(v, fn)
}

func Keep2[T any](v **T, fn VFn[T]) MFn {
	return func(pos Pos) (MPos, error) {
		return mKeep2(pos, v, fn)
	}
}

func Append[T any](w *[]T, fn VFn[T]) MFn {
	return func(pos Pos) (MPos, error) {
		return mAppend(pos, w, fn)
	}
}

func VAppend[T any](fn VFn[T]) VFn[[]T] {
	return func(pos Pos) ([]T, MPos, error) {
		return mvAppend(pos, fn)
	}
}
