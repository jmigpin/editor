package toolbarparser

import (
	"strings"
	"unicode"

	"github.com/jmigpin/editor/core/parseutil"
	"github.com/jmigpin/editor/util/statemach"
)

func Parse(str string) *Data {
	p := &Parser{}
	p.data = &Data{Str: str}
	p.sm = statemach.NewString(str)
	p.parse()
	return p.data
}

type Parser struct {
	data *Data
	sm   *statemach.String
}

func (p *Parser) parse() {
	p.data.Parts = p.parseParts()
}

func (p *Parser) parseParts() []*Part {
	var parts []*Part
	for {
		part := p.parsePart()
		parts = append(parts, part)
		if p.sm.AcceptAny("|\n") { // split parts on these runes
			p.sm.Advance()
			continue
		}
		if p.sm.AcceptRune(statemach.EOS) {
			break
		}
	}
	return parts
}

func (p *Parser) parsePart() *Part {
	part := &Part{}
	part.Data = p.data

	// position
	part.Pos = p.sm.Start
	defer func() {
		p.sm.Advance()
		part.End = p.sm.Start
	}()

	first := true
	for {
		// optional space at start, but needed between args
		if p.sm.AcceptSpaceExceptNewline() {
			p.sm.Advance()
		} else if !first {
			break
		}
		first = false

		arg, ok := p.parseArg()
		if ok {
			part.Args = append(part.Args, arg)
		}
	}

	return part
}

func (p *Parser) parseArg() (*Arg, bool) {
	arg := &Arg{}
	arg.Data = p.data

	// position
	arg.Pos = p.sm.Start
	defer func() {
		p.sm.Advance()
		arg.End = p.sm.Start
	}()

	acc := p.sm.AcceptLoopFn(func(ru rune) bool {
		switch ru {
		case statemach.EOS, '|':
			return false
		}
		if unicode.IsSpace(ru) {
			return false
		}
		if p.sm.IsQuoteAccept(ru, parseutil.QuoteRunes, parseutil.EscapeRunes) {
			return true
		}
		if p.sm.IsEscapeAccept(ru, parseutil.EscapeRunes) {
			return true
		}
		return true
	})
	if !acc {
		// empty arg. example of string with parts with empty args: "|||".
		return nil, false
	}

	return arg, true
}

//----------

type Data struct {
	Str   string
	Parts []*Part
}

func (d *Data) PartAtIndex(i int) (*Part, bool) {
	for _, p := range d.Parts {
		if i >= p.Pos && i <= p.End { // end includes separator and eos
			return p, true
		}
	}
	return nil, false
}
func (d *Data) Part0Arg0() (*Arg, bool) {
	if len(d.Parts) > 0 && len(d.Parts[0].Args) > 0 {
		return d.Parts[0].Args[0], true
	}
	return nil, false
}

//----------

type Part struct {
	Node
	Args []*Arg
}

func (p *Part) ArgsUnquoted() []string {
	args := []string{}
	for _, a := range p.Args {
		args = append(args, a.UnquotedStr())
	}
	return args
}

//----------

type Arg struct {
	Node
}

//----------

type Node struct {
	Pos  int
	End  int   // end pos
	Data *Data // data with full str
}

func (node *Node) Str() string {
	return node.Data.Str[node.Pos:node.End]
}
func (node *Node) UnquotedStr() string {
	s := node.Str()
	if len(s) >= 2 {
		if s[0] == s[len(s)-1] {
			if strings.ContainsRune(parseutil.QuoteRunes, rune(s[0])) {
				return s[1 : len(s)-1]
			}
		}
	}
	return s
}
