package btparser

// Internal wrapper helpers used to build parser rules without depending on the generated W* API.

func vByte() VFn[byte] {
	return func(ps *ParserState, pos Pos) (byte, MPos, error) {
		return mvByte(ps, pos)
	}
}

func vRune() VFn[rune] {
	return func(ps *ParserState, pos Pos) (rune, MPos, error) {
		return mvRune(ps, pos)
	}
}

func vLastRune() VFn[rune] {
	return func(ps *ParserState, pos Pos) (rune, MPos, error) {
		return mvLastRune(ps, pos)
	}
}

func token(ignore MFn, fn MFn) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mToken(ps, pos, ignore, fn)
	}
}

func and(fns ...MFn) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mAnd(ps, pos, fns...)
	}
}

func reverseAnd(fns ...MFn) MFn {
	r := make([]MFn, len(fns))
	for i := range fns {
		r[i] = fns[len(fns)-1-i]
	}
	return and(r...)
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

func limitSourceBytes(back, forward int, fn MFn) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mLimitSourceBytes(ps, pos, back, forward, fn)
	}
}

func limitSourceLines(back, forward int, fn MFn) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mLimitSourceLines(ps, pos, back, forward, fn)
	}
}

func reverseSource(fn MFn) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mReverseSource(ps, pos, fn)
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

func runeAnyOf(rus ...rune) MFn {
	m := map[rune]struct{}{}
	for _, ru := range rus {
		m[ru] = struct{}{}
	}
	return runeInMap(m)
}

func runeInMap(m map[rune]struct{}) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mRuneFn(ps, pos, func(ru rune) bool {
			_, ok := m[ru]
			return ok
		})
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

func seqOrMid(s string) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mSeqOrMid(ps, pos, s)
	}
}

func loop1(fn MFn) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mLoop1(ps, pos, fn)
	}
}

func loopStartEnd(startFn, consumeFn, endFn MFn) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		return mLoopStartEnd(ps, pos, startFn, consumeFn, endFn)
	}
}

func sourceStr(fn MFn) VFn[string] {
	return func(ps *ParserState, pos Pos) (string, MPos, error) {
		return mvSourceStr(ps, pos, fn)
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
