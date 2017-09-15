package toolbardata

import (
	"fmt"
	"strings"
	"unicode"
)

type Part struct {
	*Token
	Args []*Token
}

// "" gives nil parts.
// " " gives 1 part with zero args.
// "|" gives 1 part with zero args.
// "| " gives 2 parts with zero args each.
func parseParts(str string) []*Part {
	// split '|'
	toks := parseTokens(str, func(ru rune) bool {
		return ru == '|'
	})
	// build
	var res []*Part
	for _, tok := range toks {
		s := tok.Str

		// parse args
		args := parseTokens(s, func(ru rune) bool {
			return unicode.IsSpace(ru)
		})

		// clear args that contain only spaces
		var a []*Token
		for _, t := range args {
			if strings.TrimSpace(t.Str) == "" {
				continue
			}
			a = append(a, t)
		}
		args = a

		part := &Part{tok, args}
		res = append(res, part)
	}
	return res
}
func (p *Part) JoinArgsIndexes(s, e int) *Token {
	args := p.Args[s:e]
	tok := &Token{}
	if len(args) > 0 {
		s0 := args[0].Start
		e0 := args[len(args)-1].End
		str := p.Str[s0:e0]
		tok = &Token{Str: str, Start: s0, End: e0}
	}
	return tok
}
func (p *Part) JoinArgs() *Token {
	return p.JoinArgsIndexes(0, len(p.Args))
}
func (p *Part) JoinArgsFromIndex(s int) *Token {
	return p.JoinArgsIndexes(s, len(p.Args))
}
func (p *Part) String() string {
	var u []string
	for _, a := range p.Args {
		u = append(u, fmt.Sprintf("%v", a))
	}
	s := fmt.Sprintf("{%v [%v]}", *p.Token, strings.Join(u, " "))
	return s
}
func (p *Part) ReplaceArg(i int, str string) string {
	return p.Args[i].Replace(p.Str, str)
}
