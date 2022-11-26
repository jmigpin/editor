package toolbarparser

import (
	"sync"

	"github.com/jmigpin/editor/util/parseutil"
	"github.com/jmigpin/editor/util/parseutil/lrparser"
)

func parseVarDecl(str string) (*Var, error) {
	p, err := getVarDeclParser()
	if err != nil {
		return nil, err
	}
	return p.parseVarDecl(str)
}

//----------

// parser3 singleton
var p3s struct {
	once sync.Once
	p    *varDeclParser
	err  error
}

func getVarDeclParser() (*varDeclParser, error) {
	p3s.once.Do(func() {
		p3s.p, p3s.err = newParser3()
	})
	return p3s.p, p3s.err
}

//----------
//----------
//----------

type varDeclParser struct {
	lrp    *lrparser.Lrparser
	partsp *lrparser.ContentParser
	varsp  *lrparser.ContentParser

	esc rune // defined in grammar
}

func newParser3() (*varDeclParser, error) {
	p := &varDeclParser{}

	gram := `
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
		esc = ("\\")%;
	`

	// parse grammar
	fset := lrparser.NewFileSetFromBytes([]byte(gram))
	lrp, err := lrparser.NewLrparser(fset)
	if err != nil {
		return nil, err
	}
	p.lrp = lrp

	// build content parser
	opt2 := &lrparser.CpOpt{StartRule: "varDecl"}
	//opt2.VerboseError = true // DEBUG: slow!
	if cp, err := lrp.ContentParser(opt2); err != nil {
		return nil, err
	} else {
		p.varsp = cp
	}
	// keep escape used for unescape in var parse
	p.esc = []rune(p.lrp.MustGetStringRule("esc"))[0]
	// setup content parser ast funcs
	poe(p.varsp.SetBuildNodeFn("varDecl", p.buildVarDecl))
	poe(p.varsp.SetBuildNodeFn("tildeVar", p.buildTildeVar))
	poe(p.varsp.SetBuildNodeFn("dollarVar", p.buildDollarVar))
	poe(p.varsp.SetBuildNodeFn("varValue", p.buildVarValue))

	return p, nil
}

//----------

func (p *varDeclParser) parseVarDecl(src string) (*Var, error) {
	bnd, _, err := p.varsp.Parse([]byte(src), 0)
	if err != nil {
		return nil, err
	}
	return bnd.Data().(*Var), nil
}

//----------

func (p *varDeclParser) buildVarDecl(d *lrparser.BuildNodeData) error {
	//d.PrintRuleTree(5)
	d.SetData(d.Child(0).Data())
	return nil
}
func (p *varDeclParser) buildTildeVar(d *lrparser.BuildNodeData) error {
	//d.PrintRuleTree(5)
	v := &Var{Name: d.ChildStr(0), Value: d.Child(2).Data().(string)}
	d.SetData(v)
	return nil
}
func (p *varDeclParser) buildDollarVar(d *lrparser.BuildNodeData) error {
	//d.PrintRuleTree(5)
	v := &Var{Name: d.ChildStr(0)}
	if d2 := d.Child(1); !d2.IsEmpty() {
		v.Value = d2.Child(1).Data().(string)
	}
	d.SetData(v)
	return nil
}
func (p *varDeclParser) buildVarValue(d *lrparser.BuildNodeData) error {
	//d.PrintRuleTree(5)
	str := d.ChildStr(0)

	// TODO: should unquote? or just provide var.unquote()?
	// TODO: should not unquote, in case some var content refers to other variables, and unquoting will lose the info if it was inside a string and the "$" was escaped
	if u, err := parseutil.UnquoteString(str, p.esc); err == nil {
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
	n.SetPos(bnd.Pos(), bnd.End())
	n.Data = bnd.ExternalData().(*Data)
	return n
}

//----------

// panic on error
func poe(err error) {
	if err != nil {
		panic(err)
	}
}

//----------
//----------
//----------

// OLD CODE

//func ParseVar(str string) (*Var, error) {
//	//// TESTING
//	//return parseVar3_basedOnLrparser(str)

//	rd := iorw.NewStringReaderAt(str)
//	sc := scanutil.NewScanner(rd)
//	ru := sc.PeekRune()
//	switch ru {
//	case '~':
//		return parseTildeVar(sc)
//	case '$':
//		return parseDollarVar(sc)
//	}
//	return nil, fmt.Errorf("unexpected rune: %v", ru)
//}

////----------

//func parseTildeVar(sc *scanutil.Scanner) (*Var, error) {
//	// name
//	if !sc.Match.Sequence("~") {
//		return nil, sc.Errorf("name")
//	}
//	if !sc.Match.Int() {
//		return nil, sc.Errorf("name")
//	}
//	name := sc.Value()
//	sc.Advance()
//	// assign (must have)
//	if !sc.Match.Any("=") {
//		return nil, sc.Errorf("assign")
//	}
//	sc.Advance()
//	// value (must have)
//	v, err := parseVarValue(sc, false)
//	if err != nil {
//		return nil, err
//	}
//	// end
//	_ = sc.Match.Spaces()
//	if !sc.Match.End() {
//		return nil, sc.Errorf("not at end")
//	}

//	w := &Var{Name: name, Value: v}
//	return w, nil
//}

////----------

//func parseDollarVar(sc *scanutil.Scanner) (*Var, error) {
//	// name
//	if !sc.Match.Sequence("$") {
//		return nil, sc.Errorf("name")
//	}
//	if !sc.Match.Id() {
//		return nil, sc.Errorf("name")
//	}
//	name := sc.Value()
//	sc.Advance()

//	w := &Var{Name: name}

//	// assign (optional)
//	if !sc.Match.Any("=") {
//		return w, nil
//	}
//	sc.Advance()
//	// value (optional)
//	value, err := parseVarValue(sc, true)
//	if err != nil {
//		return nil, err
//	}
//	w.Value = value
//	// end
//	_ = sc.Match.Spaces()
//	if !sc.Match.End() {
//		return nil, sc.Errorf("not at end")
//	}

//	return w, nil
//}

////----------

//func parseVarValue(sc *scanutil.Scanner, allowEmpty bool) (string, error) {
//	if sc.Match.Quoted("\"'", osutil.EscapeRune, true, 1000) {
//		v := sc.Value()
//		sc.Advance()
//		u, err := strconv.Unquote(v)
//		if err != nil {
//			return "", sc.Errorf("unquote: %v", err)
//		}
//		return u, nil
//	} else {
//		if !sc.Match.ExceptUnescapedSpaces(osutil.EscapeRune) {
//			if !allowEmpty {
//				return "", sc.Errorf("value")
//			}
//		}
//		v := sc.Value()
//		sc.Advance()
//		return v, nil
//	}
//}
