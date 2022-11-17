package toolbarparser

import (
	"fmt"
	"log"
	"strings"

	"github.com/jmigpin/editor/util/parseutil/lrparser"
)

func parse3_basedOnLrparser(str string) *Data {
	p, err := newParser3()
	if err != nil {
		panic(err)
	}
	d, err := p.Parse(str)
	if err != nil {
		log.Print(err)
	}
	return d
}

//----------

type parser3 struct {
	cp *lrparser.ContentParser
}

func newParser3() (*parser3, error) {
	p := &parser3{}

	gram := `
		^parts = part (psep part)* .
		part = args | nil .
		args = (asep)* arg ((asep)+ arg)* (asep)* .
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
	opt := &lrparser.CpOpt{}
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
func (p *parser3) Parse(src string) (*Data, error) {
	data := &Data{}
	data.Str = src

	fset := lrparser.NewFileSetFromBytes([]byte(src))
	bnd, _, err := p.cp.ParseFileSet(fset, 0, data)
	if err != nil {
		return nil, err
	}
	_ = bnd
	//res := bnd.Data().(*Data)
	//return res, nil
	return data, nil
}

//----------

func (p *parser3) parseArgFn(esc rune, separators string) lrparser.PStateParseFn {
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

	data := bnd.ExternalData().(*Data)
	data.Parts = parts
	//data.bnd = bnd
	//bnd.SetData(data)
	return nil
}
func (p *parser3) buildPart(bnd *lrparser.BuildNodeData) error {
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
func (p *parser3) buildArgs(bnd *lrparser.BuildNodeData) error {
	//bnd.PrintRuleTree(5)

	args := []*Arg{}
	// first arg
	arg0 := bnd.Child(1).Data().(*Arg)
	args = append(args, arg0)
	// other args
	if err2 := bnd.ChildLoop(2, func(d3 *lrparser.BuildNodeData) error {
		args = append(args, d3.Child(1).Data().(*Arg))
		return nil
	}); err2 != nil {
		return err2
	}

	bnd.SetData(args)
	return nil
}
func (p *parser3) buildArg(d *lrparser.BuildNodeData) error {
	//d.PrintRuleTree(5)

	arg := &Arg{}
	arg.Node = *bndToNode(d)
	d.SetData(arg)
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
