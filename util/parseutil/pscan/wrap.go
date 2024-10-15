package pscan

// WARNING: DO NOT EDIT, THIS FILE WAS AUTO GENERATED

type Wrap struct {
	sc *Scanner
	M  *Match
}

func (w *Wrap) init(sc *Scanner) {
	w.sc = sc
	w.M = sc.M
}

func (w *Wrap) And(fns ...MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.And(pos, fns...)
	}
}

func (w *Wrap) AndNoReverse(fns ...MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.AndNoReverse(pos, fns...)
	}
}

func (w *Wrap) AndOptSpaces(sopt SpacesOpt, fns ...MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.AndOptSpaces(pos, sopt, fns...)
	}
}

func (w *Wrap) And2(aopt AndOpt, fns ...MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.And2(pos, aopt, fns...)
	}
}

func (w *Wrap) Or(fns ...MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.Or(pos, fns...)
	}
}

func (w *Wrap) Optional(fn MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.Optional(pos, fn)
	}
}

func (w *Wrap) Peek(fn MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.Peek(pos, fn)
	}
}

func (w *Wrap) Byte(b byte) MFn {
	return func(pos int) (int, error) {
		return w.M.Byte(pos, b)
	}
}

func (w *Wrap) ByteFn(fn func(byte) bool) MFn {
	return func(pos int) (int, error) {
		return w.M.ByteFn(pos, fn)
	}
}

func (w *Wrap) ByteFnLoop(fn func(byte) bool) MFn {
	return func(pos int) (int, error) {
		return w.M.ByteFnLoop(pos, fn)
	}
}

func (w *Wrap) ByteSequence(seq []byte) MFn {
	return func(pos int) (int, error) {
		return w.M.ByteSequence(pos, seq)
	}
}

func (w *Wrap) NBytesFn(n int, fn func(byte) bool) MFn {
	return func(pos int) (int, error) {
		return w.M.NBytesFn(pos, n, fn)
	}
}

func (w *Wrap) NBytes(n int) MFn {
	return func(pos int) (int, error) {
		return w.M.NBytes(pos, n)
	}
}

func (w *Wrap) OneByte() MFn {
	return func(pos int) (int, error) {
		return w.M.OneByte(pos)
	}
}

func (w *Wrap) Rune(ru rune) MFn {
	return func(pos int) (int, error) {
		return w.M.Rune(pos, ru)
	}
}

func (w *Wrap) RuneFn(fn func(rune) bool) MFn {
	return func(pos int) (int, error) {
		return w.M.RuneFn(pos, fn)
	}
}

func (w *Wrap) RuneFnLoop(fn func(rune) bool) MFn {
	return func(pos int) (int, error) {
		return w.M.RuneFnLoop(pos, fn)
	}
}

func (w *Wrap) RuneOneOf(rs []rune) MFn {
	return func(pos int) (int, error) {
		return w.M.RuneOneOf(pos, rs)
	}
}

func (w *Wrap) RuneNoneOf(rs []rune) MFn {
	return func(pos int) (int, error) {
		return w.M.RuneNoneOf(pos, rs)
	}
}

func (w *Wrap) RuneSequence(seq []rune) MFn {
	return func(pos int) (int, error) {
		return w.M.RuneSequence(pos, seq)
	}
}

func (w *Wrap) RuneSequenceMid(rs []rune) MFn {
	return func(pos int) (int, error) {
		return w.M.RuneSequenceMid(pos, rs)
	}
}

func (w *Wrap) NRunesFn(n int, fn func(rune) bool) MFn {
	return func(pos int) (int, error) {
		return w.M.NRunesFn(pos, n, fn)
	}
}

func (w *Wrap) NRunes(n int) MFn {
	return func(pos int) (int, error) {
		return w.M.NRunes(pos, n)
	}
}

func (w *Wrap) OneRune() MFn {
	return func(pos int) (int, error) {
		return w.M.OneRune(pos)
	}
}

func (w *Wrap) Sequence(seq string) MFn {
	return func(pos int) (int, error) {
		return w.M.Sequence(pos, seq)
	}
}

func (w *Wrap) SequenceMid(seq string) MFn {
	return func(pos int) (int, error) {
		return w.M.SequenceMid(pos, seq)
	}
}

func (w *Wrap) RuneRanges(rrs ...RuneRange) MFn {
	return func(pos int) (int, error) {
		return w.M.RuneRanges(pos, rrs...)
	}
}

func (w *Wrap) Loop(min, max int, fn MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.Loop(pos, min, max, fn)
	}
}

func (w *Wrap) LoopOneOrMore(fn MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.LoopOneOrMore(pos, fn)
	}
}

func (w *Wrap) LoopZeroOrMore(fn MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.LoopZeroOrMore(pos, fn)
	}
}

func (w *Wrap) LoopN(n int, fn MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.LoopN(pos, n, fn)
	}
}

func (w *Wrap) LoopSep(optLastSep bool, fn, sepFn MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.LoopSep(pos, optLastSep, fn, sepFn)
	}
}

func (w *Wrap) LoopStartEnd(min, max int, startFn, consumeFn, endFn MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.LoopStartEnd(pos, min, max, startFn, consumeFn, endFn)
	}
}

func (w *Wrap) LoopUntilNLOrEof(max int, includeNL bool, esc rune) MFn {
	return func(pos int) (int, error) {
		return w.M.LoopUntilNLOrEof(pos, max, includeNL, esc)
	}
}

func (w *Wrap) PtrFn(fn *MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.PtrFn(pos, fn)
	}
}

func (w *Wrap) Newline() MFn {
	return func(pos int) (int, error) {
		return w.M.Newline(pos)
	}
}

func (w *Wrap) Spaces(opt SpacesOpt) MFn {
	return func(pos int) (int, error) {
		return w.M.Spaces(pos, opt)
	}
}

func (w *Wrap) SpacesExceptNewline() MFn {
	return func(pos int) (int, error) {
		return w.M.SpacesExceptNewline(pos)
	}
}

func (w *Wrap) SpacesIncludingNewline() MFn {
	return func(pos int) (int, error) {
		return w.M.SpacesIncludingNewline(pos)
	}
}

func (w *Wrap) EmptyLine() MFn {
	return func(pos int) (int, error) {
		return w.M.EmptyLine(pos)
	}
}

func (w *Wrap) EmptyEof() MFn {
	return func(pos int) (int, error) {
		return w.M.EmptyEof(pos)
	}
}

func (w *Wrap) EmptyRestOfLine() MFn {
	return func(pos int) (int, error) {
		return w.M.EmptyRestOfLine(pos)
	}
}

func (w *Wrap) EscapeAny(escape rune) MFn {
	return func(pos int) (int, error) {
		return w.M.EscapeAny(pos, escape)
	}
}

func (w *Wrap) Section(open, close string, esc rune, failOnNewline bool, max int, eofClose bool, consumeFn MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.Section(pos, open, close, esc, failOnNewline, max, eofClose, consumeFn)
	}
}

func (w *Wrap) StringSection(openclose string, esc rune, failOnNewline bool, maxLen int, eofClose bool) MFn {
	return func(pos int) (int, error) {
		return w.M.StringSection(pos, openclose, esc, failOnNewline, maxLen, eofClose)
	}
}

func (w *Wrap) DoubleQuotedString(maxLen int) MFn {
	return func(pos int) (int, error) {
		return w.M.DoubleQuotedString(pos, maxLen)
	}
}

func (w *Wrap) QuotedString() MFn {
	return func(pos int) (int, error) {
		return w.M.QuotedString(pos)
	}
}

func (w *Wrap) QuotedString2(esc rune, maxLen1, maxLen2 int) MFn {
	return func(pos int) (int, error) {
		return w.M.QuotedString2(pos, esc, maxLen1, maxLen2)
	}
}

func (w *Wrap) Letter() MFn {
	return func(pos int) (int, error) {
		return w.M.Letter(pos)
	}
}

func (w *Wrap) Digit() MFn {
	return func(pos int) (int, error) {
		return w.M.Digit(pos)
	}
}

func (w *Wrap) Digits() MFn {
	return func(pos int) (int, error) {
		return w.M.Digits(pos)
	}
}

func (w *Wrap) Integer() MFn {
	return func(pos int) (int, error) {
		return w.M.Integer(pos)
	}
}

func (w *Wrap) sign() MFn {
	return func(pos int) (int, error) {
		return w.M.sign(pos)
	}
}

func (w *Wrap) Float() MFn {
	return func(pos int) (int, error) {
		return w.M.Float(pos)
	}
}

func (w *Wrap) FloatOrInteger() MFn {
	return func(pos int) (int, error) {
		return w.M.FloatOrInteger(pos)
	}
}

func (w *Wrap) Identifier() MFn {
	return func(pos int) (int, error) {
		return w.M.Identifier(pos)
	}
}

func (w *Wrap) LettersAndDigits() MFn {
	return func(pos int) (int, error) {
		return w.M.LettersAndDigits(pos)
	}
}

func (w *Wrap) HexBytes() MFn {
	return func(pos int) (int, error) {
		return w.M.HexBytes(pos)
	}
}

func (w *Wrap) RegexpFromStart(res string, cache bool, maxLen int) MFn {
	return func(pos int) (int, error) {
		return w.M.RegexpFromStart(pos, res, cache, maxLen)
	}
}

func (w *Wrap) RegexpFromStartCached(res string, maxLen int) MFn {
	return func(pos int) (int, error) {
		return w.M.RegexpFromStartCached(pos, res, maxLen)
	}
}

func (w *Wrap) MustErr(fn MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.MustErr(pos, fn)
	}
}

func (w *Wrap) PtrTrue(v *bool) MFn {
	return func(pos int) (int, error) {
		return w.M.PtrTrue(pos, v)
	}
}

func (w *Wrap) PtrFalse(v *bool) MFn {
	return func(pos int) (int, error) {
		return w.M.PtrFalse(pos, v)
	}
}

func (w *Wrap) StaticCondFn(v bool, tfn, ffn MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.StaticCondFn(pos, v, tfn, ffn)
	}
}

func (w *Wrap) Eof() MFn {
	return func(pos int) (int, error) {
		return w.M.Eof(pos)
	}
}

func (w *Wrap) NotEof() MFn {
	return func(pos int) (int, error) {
		return w.M.NotEof(pos)
	}
}

func (w *Wrap) ReverseMode(reverse bool, fn MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.ReverseMode(pos, reverse, fn)
	}
}

func (w *Wrap) LoopValue(min, max int, fn VFn) VFn {
	return func(pos int) (any, int, error) {
		return w.M.LoopValue(pos, min, max, fn)
	}
}

func (w *Wrap) LoopSepValue(optLastSep bool, fn VFn, sepFn MFn) VFn {
	return func(pos int) (any, int, error) {
		return w.M.LoopSepValue(pos, optLastSep, fn, sepFn)
	}
}

func (w *Wrap) AndValue(fns ...VFn) VFn {
	return func(pos int) (any, int, error) {
		return w.M.AndValue(pos, fns...)
	}
}

func (w *Wrap) AndFlexValue(fns ...any) VFn {
	return func(pos int) (any, int, error) {
		return w.M.AndFlexValue(pos, fns...)
	}
}

func (w *Wrap) OrValue(fns ...VFn) VFn {
	return func(pos int) (any, int, error) {
		return w.M.OrValue(pos, fns...)
	}
}

func (w *Wrap) OptionalValue(fn VFn) VFn {
	return func(pos int) (any, int, error) {
		return w.M.OptionalValue(pos, fn)
	}
}

func (w *Wrap) NilValue(fn MFn) VFn {
	return func(pos int) (any, int, error) {
		return w.M.NilValue(pos, fn)
	}
}

func (w *Wrap) BytesValue(fn MFn) VFn {
	return func(pos int) (any, int, error) {
		return w.M.BytesValue(pos, fn)
	}
}

func (w *Wrap) StrValue(fn MFn) VFn {
	return func(pos int) (any, int, error) {
		return w.M.StrValue(pos, fn)
	}
}

func (w *Wrap) RuneValue(fn MFn) VFn {
	return func(pos int) (any, int, error) {
		return w.M.RuneValue(pos, fn)
	}
}

func (w *Wrap) IntValue() VFn {
	return func(pos int) (any, int, error) {
		return w.M.IntValue(pos)
	}
}

func (w *Wrap) IntFnValue(fn MFn) VFn {
	return func(pos int) (any, int, error) {
		return w.M.IntFnValue(pos, fn)
	}
}

func (w *Wrap) Int64Value() VFn {
	return func(pos int) (any, int, error) {
		return w.M.Int64Value(pos)
	}
}

func (w *Wrap) Float32Value() VFn {
	return func(pos int) (any, int, error) {
		return w.M.Float32Value(pos)
	}
}

func (w *Wrap) Float64Value() VFn {
	return func(pos int) (any, int, error) {
		return w.M.Float64Value(pos)
	}
}

func (w *Wrap) Printf(f string, args ...any) MFn {
	return func(pos int) (int, error) {
		return w.M.Printf(pos, f, args...)
	}
}

func (w *Wrap) PrintfForOr(f string, args ...any) MFn {
	return func(pos int) (int, error) {
		return w.M.PrintfForOr(pos, f, args...)
	}
}

func (w *Wrap) PrintLineColAndSrc() MFn {
	return func(pos int) (int, error) {
		return w.M.PrintLineColAndSrc(pos)
	}
}

func (w *Wrap) PrintLineColAndSrcForOr() MFn {
	return func(pos int) (int, error) {
		return w.M.PrintLineColAndSrcForOr(pos)
	}
}

func (w *Wrap) PrintPosAndSrc() MFn {
	return func(pos int) (int, error) {
		return w.M.PrintPosAndSrc(pos)
	}
}

func (w *Wrap) PrintPosAndSrcForOr() MFn {
	return func(pos int) (int, error) {
		return w.M.PrintPosAndSrcForOr(pos)
	}
}

func (w *Wrap) FatalOnError(s string, fn MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.FatalOnError(pos, s, fn)
	}
}

func (w *Wrap) FailForOr(fn MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.FailForOr(pos, fn)
	}
}
