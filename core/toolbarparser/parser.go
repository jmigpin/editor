package toolbarparser

import (
	"log"
	"strconv"
	"unicode"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/scanutil"
)

//----------

type Data struct {
	Str   string
	Parts []*Part
}

func Parse(str string) *Data {
	p := &Parser{}
	p.data = &Data{Str: str}

	rd := iorw.NewStringReaderAt(str)
	p.sc = scanutil.NewScanner(rd)

	if err := p.start(); err != nil {
		log.Print(err)
	}

	return p.data
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

type Parser struct {
	data *Data
	sc   *scanutil.Scanner
}

func (p *Parser) start() error {
	parts, err := p.parts()
	if err != nil {
		return err
	}
	p.data.Parts = parts
	return nil
}

func (p *Parser) parts() ([]*Part, error) {
	var parts []*Part
	for {
		part, err := p.part()
		if err != nil {
			return nil, err
		}
		parts = append(parts, part)

		// split parts on these runes
		if p.sc.Match.Any("|\n") {
			p.sc.Advance()
			continue
		}
		if p.sc.Match.End() {
			break
		}
	}
	return parts, nil
}

func (p *Parser) part() (*Part, error) {
	part := &Part{}
	part.Data = p.data

	// position
	part.Pos = p.sc.Start
	defer func() {
		p.sc.Advance()
		part.End = p.sc.Start
	}()

	// optional space at start
	if p.sc.Match.SpacesExceptNewline() {
		p.sc.Advance()
	}

	for {
		arg, err := p.arg()
		if err != nil {
			break // end of part
		}
		part.Args = append(part.Args, arg)

		// need space between args
		if p.sc.Match.SpacesExceptNewline() {
			p.sc.Advance()
		} else {
			break
		}
	}
	return part, nil
}

func (p *Parser) arg() (*Arg, error) {
	arg := &Arg{}
	arg.Data = p.data

	// position
	arg.Pos = p.sc.Start
	defer func() {
		p.sc.Advance()
		arg.End = p.sc.Start
	}()

	ok := p.sc.RewindOnFalse(func() bool {
		for {
			if p.sc.Match.End() {
				break
			}
			if p.sc.Match.Escape(osutil.EscapeRune) {
				continue
			}
			if p.sc.Match.GoQuotes(osutil.EscapeRune, 1500, 1500) {
				continue
			}

			// split args
			ru := p.sc.PeekRune()
			if ru == '|' || unicode.IsSpace(ru) {
				break
			} else {
				_ = p.sc.ReadRune() // accept rune into arg
			}
		}
		return !p.sc.Empty()
	})
	if !ok {
		// empty arg. Ex: parts string with empty args: "|||".
		return nil, p.sc.Errorf("arg")
	}
	return arg, nil
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

func (p *Part) ArgsStrs() []string {
	args := []string{}
	for _, a := range p.Args {
		args = append(args, a.Str())
	}
	return args
}

func (p *Part) FromArgString(i int) string {
	if i >= len(p.Args) {
		return ""
	}
	a := p.Args[i:]
	n1 := a[0]
	n2 := a[len(a)-1]
	return p.Node.Data.Str[n1.Pos:n2.End]
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
	s2, err := strconv.Unquote(s)
	if err != nil {
		return s
	}
	return s2
}
