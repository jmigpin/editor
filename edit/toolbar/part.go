package toolbar

import (
	"fmt"
	"strings"
	"unicode"
)

type Part struct {
	*Token
	Args []*Token
}

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
		// a part will have at least an argument
		if len(args) == 0 {
			continue
		}
		part := &Part{tok, args}
		res = append(res, part)
	}
	return res
}
func (p *Part) JoinArgsIndexes(s, e int) *Token {
	args := p.Args[s:e]
	s0 := args[0].Start
	e0 := args[len(args)-1].End
	str := p.Str[s0:e0]
	return &Token{str, s0, e0}
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
