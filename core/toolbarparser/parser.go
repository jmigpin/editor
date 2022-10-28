package toolbarparser

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"unicode"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil/lrparser"
	"github.com/jmigpin/editor/util/scanutil"
)

//----------

type Data struct {
	Str   string
	Parts []*Part
	bnd   *lrparser.BuildNodeData
}

func Parse(str string) *Data {
	//return Parse2(str)

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
	End  int // end pos
	src  []byte
	Data *Data // data with full str
}

func (node *Node) Str() string {
	//return node.Data.Str[node.Pos:node.End]
	return string(node.src[node.Pos:node.End])
}

func (node *Node) UnquotedStr() string {
	s := node.Str()
	s2, err := strconv.Unquote(s)
	if err != nil {
		return s
	}
	return s2
}

//----------
//----------
//----------

func Parse2(str string) *Data {
	p, err := newParser2()
	if err != nil {
		panic(err)
	}
	d, err := p.Parse(str)
	if err != nil {
		panic(err)
	}
	return d
}

// ----------

type parser2 struct {
	cp *lrparser.ContentParser
}

func newParser2() (*parser2, error) {
	p := &parser2{}

	//gram := `
	//	//^parts = part (psep part)* .
	//	//part = (partItem)+ | nil .
	//	//partItem = !notPartRunes | string | esc anyrune .
	//	//notPartRunes = psep | %esc | %dquote .

	//	//----------

	//	^parts = part (psep part)* . // always at least one part
	//	part = args | nil .

	//	//args = (asep)* arg ((asep)+ arg)* (asep)* .	// conflict
	//	args = (asep)* arg args2 .
	//	args2 = (asep)+ arg args2 | (asep)+ | nil .

	//	arg = (string | argRunes)+ .
	//	argRunes = !notArgRunes | esc (anyrune|$) .
	//	notArgRunes = asep | psep | %esc | %dquote | %bquote.

	//	//----------

	//	//string = dquote (stringRunes)* dquote .
	//	//stringRunes = !notStringRunes | esc anyrune.
	//	//notStringRunes = %dquote | %esc . // concat
	//	//dquote = "\"" .

	//	string = string1 | string2 .
	//	string1 = dquote (stringRunes1|bquote)* (dquote|$) .
	//	//string2 = bquote (stringRunes2)* bquote .
	//	string2 = bquote (stringRunes1|dquote)* bquote .
	//	stringRunes1 = !notStringRunes1 | esc (anyrune|$).
	//	//stringRunes2 = !notStringRunes2 | esc anyrune.
	//	notStringRunes1 = %dquote | %bquote | %esc .
	//	//notStringRunes2 = %bquote | %esc .
	//	dquote = "\"" .
	//	bquote = "` + "`" + `" .

	//	psep = %"|\n" . // part separator
	//	asep = %" \t" . // arg separator
	//	esc = "\\" .
	//`

	gram := `
		^parts = part (psep part)* .
		part = args | nil .

		// arg rule defined externally for simplification/performance
		//args = (asep)* arg ((asep)+ arg)* (asep)* . // conflict
		//args = ((asep)* arg)+ (asep)* .
		args = (asep)* arg args2 .
		args2 = (asep)+ arg args2 | (asep)+ | nil .

		psep = %"|\n" . // part separator
		asep = %" \t" . // arg separator
	`

	// parse grammar
	fset := lrparser.NewFileSetFromBytes([]byte(gram))
	lrp, err := lrparser.NewLrparser(fset)
	if err != nil {
		return nil, err
	}

	// setup extra rules
	esc := '\\'
	psep, _ := lrp.GetStringRule("psep")
	asep, _ := lrp.GetStringRule("asep")
	lrp.SetFuncRule("arg", p.parseArgFn(esc, psep+asep))

	// build content parser
	opt := &lrparser.CPOpt{}
	//opt := &lrparser.CPOpt{ShiftOnSRConflict: true}
	cp, err := lrp.ContentParser(opt)
	if err != nil {
		return nil, err
	}
	// setup content parser ast funcs
	cp.SetBuildNodeFn("parts", p.buildParts)
	cp.SetBuildNodeFn("part", p.buildPart)
	cp.SetBuildNodeFn("args", p.buildArgs)
	cp.SetBuildNodeFn("arg", p.buildArg)
	p.cp = cp

	return p, nil
}
func (p *parser2) Parse(src string) (*Data, error) {
	bnd, err := p.cp.Parse([]byte(src), 0)
	if err != nil {
		return nil, err
	}
	res := bnd.Data().(*Data)
	return res, nil
}

//----------

func (p *parser2) parseArgFn(esc rune, separators string) lrparser.PStateParseFn {
	return func(ps *lrparser.PState) error {
		ps2 := ps.Copy()
		for {
			if err := ps2.EscapeAny(esc); err == nil {
				continue
			}
			if err := ps2.GoString2(3000, 10); err == nil {
				continue
			}
			// read rune
			ps3 := ps2.Copy()
			ru, err := ps3.ReadRune()
			if err != nil {
				break
			}
			if strings.Contains(separators, string(ru)) {
				break
			}
			// consume rune
			// solves also if it is an escape at end: "abc\\$"
			ps2.Set(ps3)
		}
		d := ps2.Pos() - ps.Pos()
		if d == 0 {
			return fmt.Errorf("empty")
		}
		ps.Set(ps2)
		return nil
	}
}

//----------

func (p *parser2) buildParts(bnd *lrparser.BuildNodeData) error {
	//bnd.PrintRuleTree(5)

	parts := []*Part{}
	// first part
	part0 := bnd.Child(0).Data().(*Part)
	parts = append(parts, part0)
	// other parts
	if d2 := bnd.Child(1); !d2.IsEmpty() {
		//d2.PrintRuleTree(5)
		l := d2.ChildsLen()
		for i := 0; i < l; i++ {
			part2 := d2.Child(i).Child(1).Data().(*Part)
			parts = append(parts, part2)
		}
	}

	data := &Data{}
	data.Str = string(bnd.FullSrc())
	data.Parts = parts
	data.bnd = bnd
	bnd.SetData(data)
	return nil
}
func (p *parser2) buildPart(bnd *lrparser.BuildNodeData) error {
	//d.PrintRuleTree(5)

	// part
	part := &Part{}
	part.Node = *bndToNode(bnd)
	if !bnd.IsEmpty() { // part itself can be nil
		d2 := bnd.Child(0)
		part.Args = d2.Data().([]*Arg)
	}
	bnd.SetData(part)
	return nil
}
func (p *parser2) buildArgs(bnd *lrparser.BuildNodeData) error {
	//bnd.PrintRuleTree(5)

	args := []*Arg{}
	// first arg
	arg0 := bnd.Child(1).Data().(*Arg)
	args = append(args, arg0)
	// more args
	if d2 := bnd.Child(2); !d2.IsEmpty() { // args2
		for d2.ChildsLen() == 3 {
			//d2.PrintRuleTree(5)
			if d3 := d2.Child(1); !d3.IsEmpty() { // arg
				args = append(args, d3.Data().(*Arg))
			}
			if d4 := d2.Child(2); !d4.IsEmpty() {
				d2 = d4
				continue
			}
			break
		}
	}
	bnd.SetData(args)
	return nil
}
func (p *parser2) buildArg(bnd *lrparser.BuildNodeData) error {
	//bnd.PrintRuleTree(5)

	arg := &Arg{}
	arg.Node = *bndToNode(bnd)
	bnd.SetData(arg)
	return nil
}

//----------
//----------
//----------

func bndToNode(bnd *lrparser.BuildNodeData) *Node {
	n := &Node{}
	n.Pos = bnd.Pos()
	n.End = bnd.End()
	n.src = bnd.FullSrc()
	return n
}
