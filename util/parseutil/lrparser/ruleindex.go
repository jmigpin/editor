package lrparser

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// unique rule index
type RuleIndex struct {
	m  map[string]*Rule // *DefRule,*FuncRule,*BoolRule
	cm map[string]ProcRuleFn

	deref struct {
		once bool
		err  error
	}
}

func newRuleIndex() *RuleIndex {
	ri := &RuleIndex{}
	ri.m = map[string]*Rule{}
	ri.cm = map[string]ProcRuleFn{}

	setSingleton := func(r *SingletonRule) {
		r2 := Rule(r)
		ri.m[r.name] = &r2 // ok to set directly, only for special rules
	}
	setFunc := func(name string, fn PStateParseFn) {
		if err := ri.setFuncRule(name, fn); err != nil {
			panic(err)
		}
	}
	setProc := func(name string, fn ProcRuleFn) {
		ri.cm[name] = fn // ok to set directly, only for special rules
	}

	// setup predefined rules
	setSingleton(nilRule)
	setSingleton(endRule)
	setSingleton(anyruneRule)
	setFunc("letter", parseLetter)
	setFunc("digit", parseDigit)
	setProc("&dropRunes", procDropRunes)
	setProc("&escapeAny", procEscapeAny)

	// (digits)+
	//setFn("digits", parseDigits) // can't define this since it will not be able to compose "digits" with "digit" and will fail the produce a correct parser
	// works correctly, but it is a non terminal and shows in ruleindex // TODO: improve
	//r2 := &ParenOneOrMoreRule{}
	//r2.setOnlyChild(ri.m["digit"])
	//if err := ri.setDefRule("digits", r2); err != nil {
	//	panic(err)
	//}

	return ri
}

//----------

func (ri *RuleIndex) set(name string, r Rule) error {
	if ri.deref.once {
		return fmt.Errorf("calling set after dereference")
	}

	// need a level on indirection to have the ruleindex.map be iterable without issues when making rules unique. Forcing rules in the index to be of these types provides that (allowing directly a stringrule would cause issues).
	switch r.(type) {
	case *DefRule, *FuncRule, *BoolRule:
	default:
		return fmt.Errorf("unexpected type to set in ruleindex: %T", r)
	}

	// don't allow reserverd words to be names
	switch name {
	case "", "rule", "if",
		endRule.id(),
		nilRule.id(),
		anyruneRule.id():
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
func (ri *RuleIndex) setFuncRule(name string, fn PStateParseFn) error {
	r := &FuncRule{name: name, fn: fn}
	return ri.set(name, r)
}

//----------

func (ri *RuleIndex) derefRules() error {
	if ri.deref.once {
		return ri.deref.err
	}
	ri.deref.once = true

	err := dereferenceRules(ri)
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

func (ri *RuleIndex) String() string {
	res := []string{}
	for _, r := range ri.sorted() {
		if r.isTerminal() { // don't print terminals
			continue
		}

		res = append(res, fmt.Sprintf("%v", r))
	}
	return strings.Join(res, "\n")
}

func (ri *RuleIndex) sorted() []Rule {
	w := []Rule{}
	for _, r := range ri.m {
		w = append(w, *r)
	}
	sortRules(w)
	return w
}

//----------
//----------
//----------

type ProcRuleFn func(Rule) (Rule, error)

//----------
//----------
//----------

func parseLetter(ps *PState) error {
	ps2 := ps.Copy()
	ru, err := ps2.ReadRune()
	if err != nil {
		return err
	}
	if !unicode.IsLetter(ru) {
		return errors.New("not a letter")
	}
	ps.Set(ps2)
	return nil
}
func parseDigit(ps *PState) error {
	ps2 := ps.Copy()
	ru, err := ps2.ReadRune()
	if err != nil {
		return err
	}
	if !unicode.IsDigit(ru) {
		return errors.New("not a digit")
	}
	ps.Set(ps2)
	return nil
}

// commented: using this won't recognize "digit" in "digits", which won't allow to parse correctly in some cases
//func parseDigits(ps *PState) error {
//	for i := 0; ; i++ {
//		ps2 := ps.copy()
//		ru, err := ps2.readRune()
//		if err != nil {
//			if i > 0 {
//				return nil
//			}
//			return err
//		}
//		if !unicode.IsDigit(ru) {
//			if i == 0 {
//				return errors.New("not a digit")
//			}
//			return nil
//		}
//		ps.set(ps2)
//	}
//}

//----------

// expects andrule composed of stringrules, and removes from the first rule all the other rules runes
func procDropRunes(r Rule) (Rule, error) {
	ar, ok := r.(*AndRule)
	if !ok {
		return nil, fmt.Errorf("expecting \"and\" rule")
	}
	if len(ar.childs()) < 2 {
		return nil, fmt.Errorf("expecting \"and\" rule with at least 2 childs")
	}
	srs := []*StringRule{}
	for _, c := range ar.childs() {
		sr, ok := ruleInnerStringRule(c, stringrOr)
		if !ok || sr.typ != stringrOr {
			return nil, fmt.Errorf("expecting stringrule type %q", stringrOr)
		}
		srs = append(srs, sr)
	}
	// join rules to remove
	m2 := map[rune]bool{}
	for i := 1; i < len(srs); i++ {
		for _, ru := range srs[i].runes {
			m2[ru] = true
		}
	}
	// remove from first rule
	rs := []rune{}
	for _, ru := range srs[0].runes {
		if m2[ru] {
			continue
		}
		rs = append(rs, ru)
	}
	sr3 := *srs[0] // copy
	sr3.runes = rs
	return &sr3, nil
}

// allows to rewind in case of failure
func procEscapeAny(r Rule) (Rule, error) {
	//sr, ok := r.(*StringRule)
	sr, ok := ruleInnerStringRule(r, stringrNone)
	if !ok {
		return nil, fmt.Errorf("expecting stringrule")
	}
	if len(sr.runes) != 1 {
		return nil, fmt.Errorf("expecting rule with one rune")
	}
	esc := sr.runes[0]
	fr := &FuncRule{}
	fr.fn = func(ps *PState) error {
		return ps.EscapeAny(esc)
	}
	return fr, nil
}
