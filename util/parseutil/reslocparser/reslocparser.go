package reslocparser

import (
	_ "embed"

	"github.com/jmigpin/editor/util/parseutil"
	"github.com/jmigpin/editor/util/parseutil/lrparser"
)

//go:embed reslocparser.gram
var resLocGrammar []byte
var resLocFilename = "reslocparser.gram" // for errors

//----------

var extraSyms = "_-~.%@&?!=#+:^" + "(){}[]<>" + "\\/" + " " // besides letters and digits
var nameSepSyms = "" +
	" " + // word separator
	"=" + // usually around filenames (ex: -arg=/a/b.txt)
	"(){}[]<>" + // usually used around filenames in various outputs
	":" // usually separating lines/cols from filenames

//----------

// resource locator parser (reminds url)
type ResLocParser struct {
	lrp   *lrparser.Lrparser
	locCp *lrparser.ContentParser
	revCp *lrparser.ContentParser

	Escape        rune
	PathSeparator rune
	ParseVolume   bool
}

func NewResLocParser() (*ResLocParser, error) {
	p := &ResLocParser{}
	p.Escape = '\\'
	p.PathSeparator = '/'

	fset := &lrparser.FileSet{Src: resLocGrammar, Filename: resLocFilename}
	lrp, err := lrparser.NewLrparser(fset)
	if err != nil {
		return nil, err
	}
	p.lrp = lrp

	return p, nil
}

// separate func to allow setting p.lrp.logfFn
func (p *ResLocParser) Init(logfFn func(f string, a ...interface{})) error {
	// panic on error
	poe := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	// setup predefined rules
	poe(p.lrp.SetBoolRule("rlParseVolume", p.ParseVolume))
	poe(p.lrp.SetStringRule("rlSep", string(p.PathSeparator)))
	poe(p.lrp.SetStringRule("rlEsc", string(p.Escape)))
	// setup extra symbols (with some removals)
	rm := nameSepSyms + string(p.Escape) + string(p.PathSeparator)
	u := parseutil.RunesExcept(extraSyms, rm)
	poe(p.lrp.SetStringRule("rlExtraSyms", u))

	revOpt := &lrparser.CPOpt{
		StartRule:         "reverse",
		EarlyStop:         true,
		ShiftOnSRConflict: true,
		LogfFn:            logfFn,
		Reverse:           true,
	}
	revCp, err := p.lrp.ContentParser(revOpt)
	if err != nil {
		return err
	}
	p.revCp = revCp

	locOpt := &lrparser.CPOpt{
		StartRule:         "location",
		EarlyStop:         true,
		ShiftOnSRConflict: true,
		LogfFn:            logfFn,
	}
	locCp, err := p.lrp.ContentParser(locOpt)
	if err != nil {
		return err
	}
	p.locCp = locCp
	p.locCp.SetBuildNodeFn("location", p.buildLocation)
	p.locCp.SetBuildNodeFn("cFile", p.buildCFile)
	p.locCp.SetBuildNodeFn("pyFile", p.buildPyFile)
	p.locCp.SetBuildNodeFn("schemeFile", p.buildSchemeFile)
	p.locCp.SetBuildNodeFn("cLineCol", p.buildCLineCol)

	return nil
}

//----------

func (p *ResLocParser) Parse(src []byte, index int) (*ResLoc, error) {
	logf := p.locCp.Opt.Logf

	// best effort to expand left
	logf("--- expand left: i=%v\n", index)
	bnd1, err := p.revCp.Parse(src, index)
	if err != nil {
		return nil, err
	}
	index = bnd1.End()
	logf("--- expand left: i=%v err=%v", index, err)

	bnd2, err := p.locCp.Parse(src, index)
	if err != nil {
		return nil, err
	}
	rl := bnd2.Data().(*ResLoc)
	return rl, nil
}

//----------

func (p *ResLocParser) buildLocation(d *lrparser.BuildNodeData) error {
	rl := d.Child(0).Data().(*ResLoc)
	rl.Escape = p.Escape
	rl.PathSep = p.PathSeparator
	rl.Bnd = d
	rl.Pos = d.Pos()
	rl.End = d.End()

	d.SetData(rl)
	return nil
}
func (p *ResLocParser) buildCFile(d *lrparser.BuildNodeData) error {
	rl := &ResLoc{}
	// filename
	rl.Path = d.ChildStr(0)
	// cLineCol
	if d2 := d.Child(1); !d2.IsNil() { // parenthesis optional
		d2 = d2.Child(0) // parenthesis
		d2 = d2.Child(0) // inner rule: cLineCol
		u := d2.Data().([]int)
		rl.Line = u[0]
		rl.Column = u[1]
	}
	d.SetData(rl)
	return nil
}
func (p *ResLocParser) buildPyFile(d *lrparser.BuildNodeData) error {
	rl := &ResLoc{}
	// filename
	rl.Path = d.Child(0).ChildStr(1)
	// digits
	if d2 := d.Child(1); !d2.IsNil() {
		//d2.PrintRuleTree(5)
		if line, err := d2.Child(0).ChildInt(1); err != nil {
			return err
		} else {
			rl.Line = line
		}
	}
	d.SetData(rl)
	return nil
}
func (p *ResLocParser) buildSchemeFile(d *lrparser.BuildNodeData) error {
	rl := &ResLoc{}
	// scheme
	rl.Scheme = d.ChildStr(0)
	// path
	rl.Path = d.ChildStr(2)
	// cLineCol
	if d2 := d.Child(3); !d2.IsNil() { // parenthesis optional
		d2 = d2.Child(0) // parenthesis
		d2 = d2.Child(0) // inner rule: cLineCol
		u := d2.Data().([]int)
		rl.Line = u[0]
		rl.Column = u[1]
	}
	d.SetData(rl)
	return nil
}
func (p *ResLocParser) buildCLineCol(d *lrparser.BuildNodeData) error {
	//d.PrintRuleTree(5)
	line, col := 0, 0
	// line
	if line2, err := d.ChildInt(1); err != nil {
		return err
	} else {
		line = line2
	}
	// column
	if d3 := d.Child(2); !d3.IsNil() { // parenthesis optional
		d3 = d3.Child(0) // parenthesis
		if col2, err := d3.ChildInt(1); err != nil {
			return err
		} else {
			col = col2
		}
	}
	d.SetData([]int{line, col})
	return nil
}
