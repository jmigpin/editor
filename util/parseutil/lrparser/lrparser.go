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

func (lrp *Lrparser) ContentParser(opt *CPOpt) (*ContentParser, error) {
	cp, err := newContentParser(opt, lrp.ri)
	if err != nil {
		return nil, lrp.fset.Error(err)
	}
	return cp, nil
}

//----------

// should avoid using because of parse order
//	func (lrp *Lrparser) SetFuncRule(name string, fn pstateParseFn) error {
//		return lrp.ri.setFuncRule(name, fn)
//	}

func (lrp *Lrparser) SetBoolRule(name string, v bool) error {
	return lrp.ri.setBoolRule(name, v)
}
func (lrp *Lrparser) SetStringRule(name string, s string) error {
	if s == "" {
		return fmt.Errorf("empty string")
	}
	r := &StringRule{}
	r.runes = []rune(s)
	return lrp.ri.setDefRule(name, r)
}
func (lrp *Lrparser) SetStringOrRule(name string, s string) error {
	if s == "" {
		return fmt.Errorf("empty string")
	}
	r := &StringOrRule{}
	r.runes = []rune(s)
	return lrp.ri.setDefRule(name, r)
}
