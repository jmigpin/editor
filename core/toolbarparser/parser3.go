package toolbarparser

import (
	"log"
	"sync"

	"github.com/jmigpin/editor/util/parseutil"
	"github.com/jmigpin/editor/util/parseutil/lrparser"
)

func parse3_basedOnLrparser(str string) *Data {
	p, err := getParser3()
	if err != nil {
		panic(err)
	}
	d, err := p.parseData(str)
	if err != nil {
		log.Print(err) // TODO: should return an error
		return d
	}
	// parse vars
	for _, part := range d.Parts {
		v, err := p.parseVarDecl(part.String())
		if err == nil {
			part.Vars = []*Var{v}
		}
	}
	return d
}
func parseVar3_basedOnLrparser(str string) (*Var, error) {
	p, err := getParser3()
	if err != nil {
		return nil, err
	}
	return p.parseVarDecl(str)
}

//----------

// parser3 singleton
var p3s struct {
	once sync.Once
	p    *parser3
	err  error
}

func getParser3() (*parser3, error) {
	p3s.once.Do(func() {
		p3s.p, p3s.err = newParser3()
	})
	return p3s.p, p3s.err
}

//----------
//----------
//----------

type parser3 struct {
	lrp    *lrparser.Lrparser
	partsp *lrparser.ContentParser
	varsp  *lrparser.ContentParser

	esc string // defined in grammar
}

func newParser3() (*parser3, error) {
	p := &parser3{}

	gram := `
		//parts = part (psep part)*;
		//part = args | nil;
		//args = (asep)* arg ((asep)+ arg)* (asep)*;
		
		parts = part (psep part)*;
		part = arg (asep arg)*;
		
		arg = (arg2)*;
		arg2 =
			@quotedString(1,esc,3000,3000) |
			@escapeAny(2,esc) | 
			//esc (anyrune0)? | // anyrune here matches before argrune, important otherwise anyrune will be "nil" and the nextrune will be an argrune (via negation)			
			argRune
			;
		argRune = (psep|asep)!;
		//argRune = (psep|asep|esc)!; // needed if using grammar defined escape
		//argRune = (psep|asep|quotes)!;
		//quotes = ("\"'` + "`" + `")%;
		
		esc = ("\\")%;
		psep = ("|\n")%; // part separator
		asep = (" \t")%; // arg separator
		
		//----------
		
		varDecls = varDecl (";" varDecl)*;
		varDecl = tildeVar | dollarVar;
		tildeVar = tildeName "=" varValue;
		dollarVar = dollarName ("=" varValue)?;
		tildeName = "~" ("0" | ("19")- (digits)?);
		dollarName = "$" dollarName2;
		dollarName2 = ("_"|letter) ("_-"|letter|digit)*;
		varValue = (
			@quotedString(1,esc,3000,3000) |
			@escapeAny(2,esc) | 
			@anyRune(3)
		)+;
		
		//----------
	`

	// parse grammar
	fset := lrparser.NewFileSetFromBytes([]byte(gram))
	lrp, err := lrparser.NewLrparser(fset)
	if err != nil {
		return nil, err
	}
	p.lrp = lrp

	// build content parser 1
	opt := &lrparser.CpOpt{StartRule: "parts"}
	//opt.VerboseError = true // DEBUG: slow!
	if cp, err := lrp.ContentParser(opt); err != nil {
		return nil, err
	} else {
		p.partsp = cp
	}

	// panic on error
	poe := func(err error) {
		if err != nil {
			panic(err)
		}
	}

	// setup content parser ast funcs
	poe(p.partsp.SetBuildNodeFn("parts", p.buildParts))
	poe(p.partsp.SetBuildNodeFn("part", p.buildPart))
	poe(p.partsp.SetBuildNodeFn("arg", p.buildArg))

	// build content parser 2
	opt2 := &lrparser.CpOpt{StartRule: "varDecl"}
	if cp, err := lrp.ContentParser(opt2); err != nil {
		return nil, err
	} else {
		p.varsp = cp
	}
	// setup content parser ast funcs
	poe(p.varsp.SetBuildNodeFn("varDecl", p.buildVarDecl))
	poe(p.varsp.SetBuildNodeFn("tildeVar", p.buildTildeVar))
	poe(p.varsp.SetBuildNodeFn("dollarVar", p.buildDollarVar))
	poe(p.varsp.SetBuildNodeFn("varValue", p.buildVarValue))

	// keep escape used for unescape in var parse
	p.esc = p.lrp.MustGetStringRule("esc")

	return p, nil
}

//----------

func (p *parser3) parseData(src string) (*Data, error) {
	// instantiate to pass as external data to parse
	data := &Data{}
	data.Str = src

	fset := lrparser.NewFileSetFromBytes([]byte(src))
	bnd, _, err := p.partsp.ParseFileSet(fset, 0, data)
	if err != nil {
		return data, err // NOTE: also returns data // TODO
	}
	_ = bnd
	//res := bnd.Data().(*Data)
	//return res, nil
	return data, nil
}

//----------

func (p *parser3) parseVarDecl(src string) (*Var, error) {
	bnd, _, err := p.varsp.Parse([]byte(src), 0)
	if err != nil {
		return nil, err
	}
	return bnd.Data().(*Var), nil
}

//----------

func (p *parser3) buildParts(bnd *lrparser.BuildNodeData) error {
	//bnd.PrintRuleTree(5)
	parts := []*Part{}
	// first part
	part0 := bnd.Child(0).Data().(*Part)
	parts = append(parts, part0)
	// other parts
	if err := bnd.ChildLoop(1, func(bnd2 *lrparser.BuildNodeData) error {
		// child0 is sep
		part2 := bnd2.Child(1).Data().(*Part)
		parts = append(parts, part2)
		return nil
	}); err != nil {
		return err
	}

	// use the external data (used because each node has a pointer to the *Data struct)
	data := bnd.ExternalData().(*Data)
	data.Parts = parts
	//data.bnd = bnd
	//bnd.SetData(data)
	return nil
}
func (p *parser3) buildPart(bnd *lrparser.BuildNodeData) error {
	//bnd.PrintRuleTree(5)

	args := []*Arg{}
	// first arg
	if !bnd.Child(0).IsEmpty() {
		arg0 := bnd.Child(0).Data().(*Arg)
		args = append(args, arg0)
	}
	// other args
	if err2 := bnd.ChildLoop(1, func(bnd3 *lrparser.BuildNodeData) error {
		if !bnd3.Child(1).IsEmpty() {
			args = append(args, bnd3.Child(1).Data().(*Arg))
		}
		return nil
	}); err2 != nil {
		return err2
	}

	part := &Part{Args: args}
	part.Node = *bndToNode(bnd)
	bnd.SetData(part)
	return nil
}
func (p *parser3) buildArg(bnd *lrparser.BuildNodeData) error {
	//d.PrintRuleTree(5)
	if bnd.IsEmpty() {
		return nil
	}
	arg := &Arg{}
	arg.Node = *bndToNode(bnd)
	bnd.SetData(arg)
	return nil
}

//----------

func (p *parser3) buildVarDecl(d *lrparser.BuildNodeData) error {
	//d.PrintRuleTree(5)
	d.SetData(d.Child(0).Data())
	return nil
}
func (p *parser3) buildTildeVar(d *lrparser.BuildNodeData) error {
	//d.PrintRuleTree(5)
	v := &Var{Name: d.ChildStr(0), Value: d.Child(2).Data().(string)}
	d.SetData(v)
	return nil
}
func (p *parser3) buildDollarVar(d *lrparser.BuildNodeData) error {
	//d.PrintRuleTree(5)
	v := &Var{Name: d.ChildStr(0)}
	if d2 := d.Child(1); !d2.IsEmpty() {
		v.Value = d2.Child(1).Data().(string)
	}
	d.SetData(v)
	return nil
}
func (p *parser3) buildVarValue(d *lrparser.BuildNodeData) error {
	//d.PrintRuleTree(5)
	str := d.ChildStr(0)

	// TODO: should unquote? or just provide var.unquote()?
	if u, err := parseutil.UnquoteString(str, []rune(p.esc)[0]); err == nil {
		str = u
	}

	d.SetData(str)
	return nil
}

//----------
//----------
//----------

func bndToNode(bnd *lrparser.BuildNodeData) *Node {
	n := &Node{}
	n.Pos = bnd.Pos()
	n.End = bnd.End()
	n.Data = bnd.ExternalData().(*Data)
	return n
}
