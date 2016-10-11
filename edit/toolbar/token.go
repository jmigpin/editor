package toolbar

import "strings"

type Token struct {
	Str        string
	Start, End int // indexes in parent string
}

func parseTokens(str string, stopRune func(rune) bool) []*Token {
	var res []*Token
	var tok *Token
	// states
	type State int
	var stk []State
	var normal, escape, quote State = 0, 1, 2
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
			if !stopRune(ru) {
				if tok == nil {
					tok = &Token{Start: ri}
					res = append(res, tok)
				}
			} else {
				if tok != nil {
					tok.End = ri
					tok = nil
				}
			}
			switch ru {
			case '\\':
				pushState(escape)
			case '"':
				pushState(quote)
			}
		case escape:
			// let this rune pass
			popState()
		case quote:
			switch ru {
			case '"':
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

func (tok *Token) Trim() string {
	s := tok.Str

	// remove escape char
	//s = strings.Replace(s, "\\", "", -1)
	// TODO: fix this by doing lookahead
	s = strings.Replace(s, "\\\\", "@-@=@", -1)
	s = strings.Replace(s, "\\", "", -1)
	s = strings.Replace(s, "@-@=@", "\\", -1)

	s = strings.TrimSpace(s)
	return s
}
