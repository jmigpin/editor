package toolbarparser

import (
	"log"
	"sync"

	"github.com/jmigpin/editor/util/parseutil/lrparser"
)

func parseVarDecl(str string) (*VarDecl, error) {
	p, err := getVarDeclParser()
	if err != nil {
		return nil, err
	}
	return p.parseVarDecl(str)
}
func parseVarRefs(str string) ([]*VarRef, error) {
	p, err := getVarDeclParser()
	if err != nil {
		return nil, err
	}
	return p.parseVarRefs(str)
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
	lrp      *lrparser.Lrparser
	varDeclP *lrparser.ContentParser
	varRefsP *lrparser.ContentParser

	esc rune // defined in grammar
}

func newParser3() (*varDeclParser, error) {
	p := &varDeclParser{}

	gram := `
		//varDecls = varDecl (";" varDecl)*; // TODO		
		varDecl = tildeVarDecl | dollarVarDecl;
		tildeVarDecl = tildeName "=" varValue;
		dollarVarDecl = dollarName ("=" varValue)?;
		tildeName = tilde tildeName2;
		tildeName2 = ("0" | ("19")- (digits)?); // not "~01"
		dollarName = dollar dollarName2;
		dollarName2 = ("_"|letter) ("_-"|letter|digit)*; // not "$1"
		varValue = (
			@quotedString(1,esc,3000,3000) |
			@escapeAny(2,esc) | 
			@anyRune(3)
		)+;
		tilde = "~";
		dollar = "$";
		esc = ("\\")%;
		
		//----------
		
		varRefs = (
			@quotedString(1,esc,3000,3000) |
			@escapeAny(2,esc) |			
			varRef |
			anyRuneLast | // allow other runes
			(tilde|dollar) // allow varrefs fails to continue
		)*;
		varRef = tildeVarRef | dollarVarRef;
		tildeVarRef = tilde (tildeName2 | "{" tildeName2 "}"); 		
		dollarVarRef = dollar (dollarName2 | "{" dollarName2 "}");
	`

	// parse grammar
	fset := lrparser.NewFileSetFromBytes([]byte(gram))
	lrp, err := lrparser.NewLrparser(fset)
	if err != nil {
		return nil, err
	}
	p.lrp = lrp

	// build content parser 1
	opt := &lrparser.CpOpt{StartRule: "varDecl"}
	//opt.VerboseError = true // DEBUG: slow!
	if cp, err := lrp.ContentParser(opt); err != nil {
		return nil, err
	} else {
		p.varDeclP = cp
	}
	// keep escape used for unescape in var parse
	p.esc = []rune(p.lrp.MustGetStringRule("esc"))[0]
	// setup content parser ast funcs
	poe(p.varDeclP.SetBuildNodeFn("varDecl", p.buildVarDecl))
	poe(p.varDeclP.SetBuildNodeFn("tildeVarDecl", p.buildTildeVar))
	poe(p.varDeclP.SetBuildNodeFn("dollarVarDecl", p.buildDollarVar))
	poe(p.varDeclP.SetBuildNodeFn("varValue", p.buildVarValue))

	// build content parser 2
	opt2 := &lrparser.CpOpt{StartRule: "varRefs"}
	//opt2.VerboseError = true // DEBUG: slow!
	if cp, err := lrp.ContentParser(opt2); err != nil {
		return nil, err
	} else {
		p.varRefsP = cp
	}
	// setup content parser ast funcs
	poe(p.varRefsP.SetBuildNodeFn("varRefs", p.buildVarRefs))
	poe(p.varRefsP.SetBuildNodeFn("varRef", p.buildVarRef))
	poe(p.varRefsP.SetBuildNodeFn("tildeVarRef", p.buildTildeVarRef))
	poe(p.varRefsP.SetBuildNodeFn("dollarVarRef", p.buildDollarVarRef))

	return p, nil
}

//----------

func (p *varDeclParser) parseVarDecl(src string) (*VarDecl, error) {
	bnd, _, err := p.varDeclP.Parse([]byte(src), 0)
	if err != nil {
		return nil, err
	}
	return bnd.Data().(*VarDecl), nil
}

//----------

func (p *varDeclParser) buildVarDecl(d *lrparser.BuildNodeData) error {
	//d.PrintRuleTree(5)
	d.SetData(d.Child(0).Data())
	return nil
}
func (p *varDeclParser) buildTildeVar(d *lrparser.BuildNodeData) error {
	v := &VarDecl{Name: d.ChildStr(0), Value: d.Child(2).Data().(string)}
	d.SetData(v)
	return nil
}
func (p *varDeclParser) buildDollarVar(d *lrparser.BuildNodeData) error {
	v := &VarDecl{Name: d.ChildStr(0)}
	if d2 := d.Child(1); !d2.IsEmpty() {
		v.Value = d2.Child(1).Data().(string)
	}
	d.SetData(v)
	return nil
}
func (p *varDeclParser) buildVarValue(d *lrparser.BuildNodeData) error {
	str := d.ChildStr(0)
	d.SetData(str)
	return nil
}

//----------

func (p *varDeclParser) parseVarRefs(src string) ([]*VarRef, error) {
	bnd, _, err := p.varRefsP.Parse([]byte(src), 0)
	if err != nil {
		return nil, err
	}
	return bnd.Data().([]*VarRef), nil
}

//----------

func (p *varDeclParser) buildVarRefs(d *lrparser.BuildNodeData) error {
	//d.PrintRuleTree(5)
	refs := []*VarRef{}
	if err := d.ChildLoop(0, func(d2 *lrparser.BuildNodeData) error {
		if u, ok := d2.Child(0).Data().(*VarRef); ok {
			refs = append(refs, u)
		}
		return nil
	}); err != nil {
		return err
	}
	d.SetData(refs)
	return nil
}

func (p *varDeclParser) buildVarRef(d *lrparser.BuildNodeData) error {
	d.SetData(d.Child(0).Data())
	return nil
}
func (p *varDeclParser) buildTildeVarRef(d *lrparser.BuildNodeData) error {
	v := &VarRef{Name: "~"}
	v.SetPos(d.Pos(), d.End())
	d2 := d.Child(1)
	switch {
	case d2.IsOr(0):
		v.Name += d2.ChildStr(0)
	case d2.IsOr(1):
		v.Name += d2.ChildStr(1)
	default:
		panic("!")
	}
	d.SetData(v)
	return nil
}
func (p *varDeclParser) buildDollarVarRef(d *lrparser.BuildNodeData) error {
	v := &VarRef{Name: "$"}
	v.SetPos(d.Pos(), d.End())
	d2 := d.Child(1)
	switch {
	case d2.IsOr(0):
		v.Name += d2.ChildStr(0)
	case d2.IsOr(1):
		v.Name += d2.ChildStr(1)
	default:
		panic("!")
	}
	d.SetData(v)
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

func expandVarRefs(src string, mapping func(string) (string, bool)) string {
	refs, err := parseVarRefs(src)
	if err != nil {
		log.Println(err)
		return src
	}
	adjust := 0
	for _, vr := range refs {
		v, ok := mapping(vr.Name)
		if !ok {
			continue
		}
		// replace: refs are expected to be in ascending order
		pos := vr.Pos() + adjust
		end := vr.End() + adjust
		src = src[0:pos] + v + src[end:]
		adjust += len(v) - (end - pos)
	}
	return src
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
