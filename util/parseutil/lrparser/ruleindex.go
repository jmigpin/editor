package lrparser

import (
	"fmt"
	"strings"
)

// unique rule index
type RuleIndex struct {
	m  map[string]*Rule
	pm map[string]ProcRuleFn

	deref struct {
		once bool
		err  error
	}
}

func newRuleIndex() *RuleIndex {
	ri := &RuleIndex{}
	ri.m = map[string]*Rule{}
	ri.pm = map[string]ProcRuleFn{}
	return ri
}

//----------

func (ri *RuleIndex) set(name string, r Rule) error {
	if ri.deref.once {
		return fmt.Errorf("calling set after dereference")
	}

	// need a level on indirection to have the ruleindex.map be iterable without issues when making rules unique. Forcing rules in the index to be of these types provides that (allowing directly a stringrule would cause issues).
	switch r.(type) {
	case *DefRule, *FuncRule, *BoolRule, *SingletonRule:
	default:
		return fmt.Errorf("unexpected type to set in ruleindex: %T", r)
	}

	// don't allow reserverd words to be names
	switch name {
	case "", "rule", "if":
		return fmt.Errorf("bad rule name: %q", name)
	}

	if ri.has(name) {
		return fmt.Errorf("rule already set: %v", name)
	}

	ri.m[name] = &r
	return nil
}
func (ri *RuleIndex) has(name string) bool {
	_, ok := ri.m[name]
	return ok
}
func (ri *RuleIndex) get(name string) (Rule, bool) {
	r, ok := ri.m[name]
	if ok {
		return *r, true
	}
	return nil, false
}
func (ri *RuleIndex) delete(name string) {
	delete(ri.m, name)
}

//----------

func (ri *RuleIndex) setDefRule(name string, r Rule) error {
	r2 := &DefRule{name: name}
	r2.setOnlyChild(r)
	return ri.set(name, r2)
}
func (ri *RuleIndex) setBoolRule(name string, v bool) error {
	r := &BoolRule{name: name, value: v}
	return ri.set(name, r)
}
func (ri *RuleIndex) setFuncRule(name string, parseOrder int, fn PStateParseFn) error {
	r := &FuncRule{name: name, parseOrder: parseOrder, fn: fn}
	return ri.set(name, r)
}
func (ri *RuleIndex) setSingletonRule(r *SingletonRule) error {
	return ri.set(r.name, r)
}

//----------

func (ri *RuleIndex) setProcRuleFn(name string, fn ProcRuleFn) error {
	if _, ok := ri.pm[name]; ok {
		return fmt.Errorf("already defined: %v", name)
	}
	ri.pm[name] = fn
	return nil
}

//----------

func (ri *RuleIndex) derefRules() error {
	if ri.deref.once {
		return ri.deref.err
	}
	err := dereferenceRules(ri)
	ri.deref.once = true
	ri.deref.err = err
	return err
}

//----------

func (ri *RuleIndex) startRule(name string) (*DefRule, error) {
	if r, ok := ri.m[name]; ok {
		dr, ok := (*r).(*DefRule)
		if !ok {
			return nil, fmt.Errorf("not a defrule: %v", r)
		}
		dr.isStart = true
		return dr, nil
	}
	// auto find marked rule
	if name == "" {
		res := (*DefRule)(nil)
		for _, r := range ri.sorted() {
			if dr, ok := r.(*DefRule); ok {
				if dr.isStart {
					if res != nil {
						return nil, fmt.Errorf("no rule name given and more then one start rule defined")
					}
					res = dr
				}
			}
		}
		if res != nil {
			return res, nil
		}
	}
	return nil, fmt.Errorf("start rule not found: %q", name)
}

//----------

//godebug:annotateoff
func (ri *RuleIndex) String() string {
	res := []string{}
	for _, r := range ri.sorted() {
		if r.isTerminal() { // don't print terminals
			continue
		}
		if dr, ok := r.(*DefRule); ok && dr.isNoPrint {
			continue
		}
		res = append(res, fmt.Sprintf("%v", r))
	}
	return fmt.Sprintf("ruleindex{\n\t%v\n}", strings.Join(res, "\n\t"))
}

func (ri *RuleIndex) sorted() []Rule {
	w := []Rule{}
	for _, r := range ri.m {
		w = append(w, *r)
	}
	sortRules(w)
	return w
}
