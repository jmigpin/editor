package lrparser

import "fmt"

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
	gp := newGrammarParser()
	ri, err := gp.parse(lrp.fset) // creates ruleindex (has some predefined rules)
	if err != nil {
		return nil, err
	}
	lrp.ri = ri
	return lrp, nil
}

func (lrp *Lrparser) ContentParser(opt *CpOpt) (*ContentParser, error) {
	cp, err := newContentParser(opt, lrp.ri)
	if err != nil {
		return nil, lrp.fset.Error(err)
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
		return "", fmt.Errorf("not a stringrule: %v", r)
	}
	return string(sr.runes), nil
}

func (lrp *Lrparser) SetBoolRule(name string, v bool) error {
	return lrp.ri.setBoolRule(name, v)
}

// NOTE: avoid using setFuncRule accessible because of parse order
func (lrp *Lrparser) SetFuncRule(name string, fn PStateParseFn) error {
	return lrp.ri.setFuncRule(name, fn)
}
