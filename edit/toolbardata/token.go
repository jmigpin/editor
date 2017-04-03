package toolbardata

import "strconv"

type Token struct {
	Start, End int    // indexes in parent string
	Str        string // result of parent string [Start:End]
}

func parseTokens(str string, stopRune func(rune) bool) []*Token {
	var res []*Token
	var tok *Token

	// states
	type State int
	var stk []State

	var normal, escape, dquote, bquote State = 0, 1, 2, 3
	pushState := func(v State) {
		stk = append(stk, v)
	}
	popState := func() {
		if len(stk) == 0 {
			panic("!")
		}
		stk = stk[:len(stk)-1]
	}
	peekState := func() State {
		return stk[len(stk)-1]
	}

	// parse
	pushState(normal)
	for ri, ru := range str {
		switch peekState() {
		case normal:
			if tok == nil {
				tok = &Token{Start: ri}
				res = append(res, tok)
			}
			if stopRune(ru) {
				tok.End = ri
				tok = nil
			}

			switch ru {
			case '\\':
				pushState(escape)
			case '"':
				pushState(dquote)
			case '`':
				pushState(bquote)
			}
		case escape:
			// let this rune pass
			popState()
		case dquote:
			switch ru {
			case '"':
				popState()
			case '\\':
				pushState(escape)
			}
		case bquote:
			switch ru {
			case '`':
				popState()
			case '\\':
				pushState(escape)
			}
		}
	}
	if tok != nil {
		tok.End = len(str)
	}
	// set tokens strings
	for _, t := range res {
		t.Str = str[t.Start:t.End]
	}
	return res
}
func (tok *Token) Replace(parentS, newS string) string {
	s, e := tok.Start, tok.End
	return parentS[:s] + newS + parentS[e:]
}
func (tok *Token) Unquote() string {
	s, err := strconv.Unquote(tok.Str)
	if err != nil {
		return tok.Str
	}
	return s
}
