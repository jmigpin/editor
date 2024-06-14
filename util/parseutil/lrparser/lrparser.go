package lrparser

import (
	"fmt"
)

////godebug:annotatepackage:github.com/jmigpin/editor/util/parseutil

type Lrparser struct {
	fset *FileSet
	ri   *RuleIndex
}

func NewLrparserFromBytes(src []byte) (*Lrparser, error) {
	fset := NewFileSetFromBytes(src)
	return NewLrparser(fset)
}
func NewLrparserFromString(src string) (*Lrparser, error) {
	return NewLrparserFromBytes([]byte(src))
}
func NewLrparser(fset *FileSet) (*Lrparser, error) {
	lrp := &Lrparser{fset: fset}
	// rule index (setups predefined rules)
	lrp.ri = newRuleIndex()
	if err := setupPredefineds(lrp.ri); err != nil {
		return nil, err
	}
	// parse provided grammar
	gp := newGrammarParser(lrp.ri)
	if err := gp.parse(lrp.fset); err != nil {
		return nil, err
	}
	return lrp, nil
}

func (lrp *Lrparser) ContentParser(opt *CpOpt) (*ContentParser, error) {
	cp, err := newContentParser(opt, lrp.ri)
	if err != nil {
		err = lrp.fset.Error(err) // attempt at improving error; these are ruleindex errors
		return nil, err
	}
	return cp, nil
}

//----------

func (lrp *Lrparser) SetStringRule(name string, s string) error {
	if s == "" {
		return fmt.Errorf("empty string")
	}
	r := &StringRule{}
	r.runes = []rune(s)
	return lrp.ri.setDefRule(name, r)
}
func (lrp *Lrparser) SetBoolRule(name string, v bool) error {
	return lrp.ri.setBoolRule(name, v)
}
func (lrp *Lrparser) SetFuncRule(name string, parseOrder int, fn PStateParseFn) error {
	return lrp.ri.setFuncRule(name, parseOrder, fn)
}

//----------

func (lrp *Lrparser) MustGetStringRule(name string) string {
	if s, err := lrp.GetStringRule(name); err != nil {
		panic(err)
	} else {
		return s
	}
}
func (lrp *Lrparser) GetStringRule(name string) (string, error) {
	r, ok := lrp.ri.get(name)
	if !ok {
		return "", fmt.Errorf("rule not found: %v", name)
	}
	dr, ok := r.(*DefRule)
	if !ok {
		return "", fmt.Errorf("not a defrule: %v", r)
	}
	sr, ok := dr.onlyChild().(*StringRule)
	if !ok {
		return "", fmt.Errorf("expecting stringrule: %v", dr.onlyChild())
	}
	return string(sr.runes), nil
}
