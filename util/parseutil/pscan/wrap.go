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

func (w *Wrap) AndR(fns ...MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.AndR(pos, fns...)
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

func (w *Wrap) LimitedLoop(min, max int, fn MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.LimitedLoop(pos, min, max, fn)
	}
}

func (w *Wrap) Loop(fn MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.Loop(pos, fn)
	}
}

func (w *Wrap) OptLoop(fn MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.OptLoop(pos, fn)
	}
}

func (w *Wrap) NLoop(n int, fn MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.NLoop(pos, n, fn)
	}
}

func (w *Wrap) loopSep0(fn, sep MFn, lastSep bool) MFn {
	return func(pos int) (int, error) {
		return w.M.loopSep0(pos, fn, sep, lastSep)
	}
}

func (w *Wrap) LoopSep(fn, sep MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.LoopSep(pos, fn, sep)
	}
}

func (w *Wrap) LoopSepCanHaveLast(fn, sep MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.LoopSepCanHaveLast(pos, fn, sep)
	}
}

func (w *Wrap) PtrFn(fn *MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.PtrFn(pos, fn)
	}
}

func (w *Wrap) Spaces(includeNL bool, escape rune) MFn {
	return func(pos int) (int, error) {
		return w.M.Spaces(pos, includeNL, escape)
	}
}

func (w *Wrap) EscapeAny(escape rune) MFn {
	return func(pos int) (int, error) {
		return w.M.EscapeAny(pos, escape)
	}
}

func (w *Wrap) ToNLOrErr(includeNL bool, esc rune) MFn {
	return func(pos int) (int, error) {
		return w.M.ToNLOrErr(pos, includeNL, esc)
	}
}

func (w *Wrap) Section(open, close string, esc rune, failOnNewline bool, maxLen int, eofClose bool, consumeFn MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.Section(pos, open, close, esc, failOnNewline, maxLen, eofClose, consumeFn)
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

func (w *Wrap) StaticTrue(v bool) MFn {
	return func(pos int) (int, error) {
		return w.M.StaticTrue(pos, v)
	}
}

func (w *Wrap) StaticFalse(v bool) MFn {
	return func(pos int) (int, error) {
		return w.M.StaticFalse(pos, v)
	}
}

func (w *Wrap) FnTrue(fn func() bool) MFn {
	return func(pos int) (int, error) {
		return w.M.FnTrue(pos, fn)
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

func (w *Wrap) OnValue(fn VFn, cb func(any)) MFn {
	return func(pos int) (int, error) {
		return w.M.OnValue(pos, fn, cb)
	}
}

func (w *Wrap) OnValue2(fn VFn, cb func(any) error) MFn {
	return func(pos int) (int, error) {
		return w.M.OnValue2(pos, fn, cb)
	}
}

func (w *Wrap) OrValue(fns ...VFn) VFn {
	return func(pos int) (any, int, error) {
		return w.M.OrValue(pos, fns...)
	}
}

func (w *Wrap) BytesValue(fn MFn) VFn {
	return func(pos int) (any, int, error) {
		return w.M.BytesValue(pos, fn)
	}
}

func (w *Wrap) StringValue(fn MFn) VFn {
	return func(pos int) (any, int, error) {
		return w.M.StringValue(pos, fn)
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

func (w *Wrap) Float64Value() VFn {
	return func(pos int) (any, int, error) {
		return w.M.Float64Value(pos)
	}
}

func (w *Wrap) PrintfNoErr(f string, args ...any) MFn {
	return func(pos int) (int, error) {
		return w.M.PrintfNoErr(pos, f, args...)
	}
}

func (w *Wrap) PrintfErr(f string, args ...any) MFn {
	return func(pos int) (int, error) {
		return w.M.PrintfErr(pos, f, args...)
	}
}

func (w *Wrap) FatalOnError(s string, fn MFn) MFn {
	return func(pos int) (int, error) {
		return w.M.FatalOnError(pos, s, fn)
	}
}
