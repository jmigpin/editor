package btparser

// Internal wrapper helpers used to build parser rules without depending on the generated W* API.

func (p *Parser) vByte() VFn[byte] {
	return func(pos Pos) (byte, MPos, error) {
		return p.mvByte(pos)
	}
}

func (p *Parser) vRune() VFn[rune] {
	return func(pos Pos) (rune, MPos, error) {
		return p.mvRune(pos)
	}
}

func (p *Parser) vLastRune() VFn[rune] {
	return func(pos Pos) (rune, MPos, error) {
		return p.mvLastRune(pos)
	}
}

func (p *Parser) token(fn MFn) MFn {
	return func(pos Pos) (MPos, error) {
		return p.mToken(pos, fn)
	}
}

func (p *Parser) and(fns ...MFn) MFn {
	return func(pos Pos) (MPos, error) {
		return p.mAnd(pos, fns...)
	}
}

func (p *Parser) or(fns ...MFn) MFn {
	return func(pos Pos) (MPos, error) {
		return p.mOr(pos, fns...)
	}
}

func (p *Parser) optional(fn MFn) MFn {
	return func(pos Pos) (MPos, error) {
		return p.mOptional(pos, fn)
	}
}

func (p *Parser) lookahead(fn MFn) MFn {
	return func(pos Pos) (MPos, error) {
		return p.mLookahead(pos, fn)
	}
}

func (p *Parser) not(fn MFn) MFn {
	return func(pos Pos) (MPos, error) {
		return p.mNot(pos, fn)
	}
}

func (p *Parser) byteFn(fn func(byte) bool) MFn {
	return func(pos Pos) (MPos, error) {
		return p.mByteFn(pos, fn)
	}
}

func (p *Parser) runeFn(fn func(rune) bool) MFn {
	return func(pos Pos) (MPos, error) {
		return p.mRuneFn(pos, fn)
	}
}

func (p *Parser) runeFnLoop(fn func(rune) bool) MFn {
	return func(pos Pos) (MPos, error) {
		return p.mRuneFnLoop(pos, fn)
	}
}

func (p *Parser) rune(ru rune) MFn {
	return func(pos Pos) (MPos, error) {
		return p.mRune(pos, ru)
	}
}

func (p *Parser) nRunes(n int) MFn {
	return func(pos Pos) (MPos, error) {
		return p.mNRunes(pos, n)
	}
}

func (p *Parser) maxNRunes(n int) MFn {
	return func(pos Pos) (MPos, error) {
		return p.mMaxNRunes(pos, n)
	}
}

func (p *Parser) seq(s string) MFn {
	return func(pos Pos) (MPos, error) {
		return p.mSeq(pos, s)
	}
}

func (p *Parser) loop1(fn MFn) MFn {
	return func(pos Pos) (MPos, error) {
		return p.mLoop1(pos, fn)
	}
}

func (p *Parser) loopStartEnd(startFn, consumeFn, endFn MFn) MFn {
	return func(pos Pos) (MPos, error) {
		return p.mLoopStartEnd(pos, startFn, consumeFn, endFn)
	}
}

func (p *Parser) sourceStr(fn MFn) VFn[string] {
	return func(pos Pos) (string, MPos, error) {
		return p.mvSourceStr(pos, fn)
	}
}

func (p *Parser) escape(esc rune) MFn {
	return func(pos Pos) (MPos, error) {
		return p.mEscape(pos, esc)
	}
}

func keep[T any](v *T, fn VFn[T]) MFn {
	return func(pos Pos) (MPos, error) {
		return mKeep(pos, v, fn)
	}
}
