package toolbar

import (
	"fmt"
	"os"
	"strings"
	"unicode"
)

type Part struct {
	*Token
	Tag  string
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
		// parse tag
		tag := ""
		tagSplitIndex := 0
		a := strings.SplitN(s, ":", 2)
		if len(a) == 2 {
			a0 := strings.TrimSpace(a[0])
			if len(a0) == 1 { // at tag has one rune
				// keep tag
				tag = a0
				tagSplitIndex = len(a[0]) + 1 // +1 is ':'
				// continue with rest of string without tag
				s = a[1]
			}
		}
		// parse args
		args := parseTokens(s, func(ru rune) bool {
			return unicode.IsSpace(ru)
		})
		// adjust args indexes
		for _, a := range args {
			a.Start += tagSplitIndex
			a.End += tagSplitIndex
		}
		// a part will have at least an argument
		if len(args) == 0 {
			continue
		}

		part := &Part{tok, tag, args}
		res = append(res, part)
	}
	//fmt.Printf("**%v\n",res)
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
func (p *Part) IsCommandTag() bool {
	//return p.Tag == "c"
	return p.Tag == ""
}

// useful if implementing the os/exec with parsed args
//func (p *Part) CommandTag() ([]string, bool) {
//if p.Tag != "c" {
//return nil, false
//}
//var cmd []string
//for _, a := range p.Args {
//s := a.Trim()
//cmd = append(cmd, s)
//}
//return cmd, true
//}
func (p *Part) String() string {
	var u []string
	for _, a := range p.Args {
		u = append(u, fmt.Sprintf("%v", a))
	}
	s := fmt.Sprintf("{%v [%v]}", *p.Token, strings.Join(u, " "))
	return s
}

func replaceHomeVar(s string) string {
	home := os.Getenv("HOME")
	if strings.HasPrefix(s, home) {
		return "~" + s[len(home):]
	}
	return s
}
func insertHomeVar(s string) string {
	if strings.HasPrefix(s, "~") {
		home := os.Getenv("HOME")
		return strings.Replace(s, "~", home, 1)
	}
	return s
}
