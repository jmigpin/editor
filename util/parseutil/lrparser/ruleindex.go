package lrparser

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/jmigpin/editor/util/goutil"
)

// unique rule index
type RuleIndex struct {
	m map[string]*Rule // *DefRule,*FuncRule,*BoolRule

	deref struct {
		once bool
		err  error
	}
}

func newRuleIndex() *RuleIndex {
	ri := &RuleIndex{m: map[string]*Rule{}}

	setSingleton := func(r *SingletonRule) {
		r2 := Rule(r)
		ri.m[r.name] = &r2 // ok to set directly, only for special rules
	}
	setFn := func(name string, fn pstateParseFn) {
		if err := ri.setFuncRule(name, fn); err != nil {
			panic(err)
		}
	}

	// setup predefined rules
	setSingleton(nilRule)
	setSingleton(endRule)
	setSingleton(anyruneRule)
	setFn("letter", parseLetter)
	setFn("digit", parseDigit)

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
	return *r, ok
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
func (ri *RuleIndex) setFuncRule(name string, fn pstateParseFn) error {
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

func dereferenceRules(ri *RuleIndex) error {
	// replace refrules first to avoid rule ids with "refs"
	if err := replaceRefRules(ri); err != nil {
		return err
	}

	if err := replaceIfRules(ri); err != nil {
		return err
	}

	// the rule index will not have parenthesis rules after this step, as they will be transformed into defrule with the equivalent id, using and/or rules
	if err := replaceParenthesisRules(ri); err != nil {
		return err
	}

	// make rules unique
	// - the pos is lost since the repeated rules are replaced with the first definition
	// - the rule src position must not be used after this function
	if err := makeRulesUnique(ri); err != nil {
		return err
	}

	return nil
}

func replaceRefRules(ri *RuleIndex) error {
	return visitRulesOnce(ri, func(rref *Rule) error {
		switch t := (*rref).(type) {
		case *RefRule:
			// replace with rule in ruleindex
			if !replaceFromMap(ri.m, t.name, rref) {
				err := fmt.Errorf("rule not found: %v", t.name)
				return &PosError{err: err, Pos: t.Pos()}
			}
			// solve stringrule references (replace with stringrule)
			if t.stringrType != stringrNone {
				t2 := *rref // just replaced above
				sr, ok := ruleInnerStringRule(t2, ri.m)
				if !ok {
					return &PosError{err: fmt.Errorf("expecting stringrule"), Pos: t.Pos()}
				}
				r4 := *sr // copy (to set type)
				r4.typ = t.stringrType
				*rref = &r4
			}
		}
		return nil
	})
}

func replaceIfRules(ri *RuleIndex) error {
	return visitRulesOnce(ri, func(rref *Rule) error {
		switch t := (*rref).(type) {
		case *IfRule:
			c0 := t.childs2[0] // conditional rule
			c1 := t.childs2[1] // rule if condition is true
			c2 := t.childs2[2] // rule if condition is false
			c0br, ok := c0.(*BoolRule)
			if !ok {
				return fmt.Errorf("ifrule condition is not a boolrule: %v (%T)", c0, c0)
			}
			// observe the value now
			if c0br.value {
				*rref = c1
			} else {
				*rref = c2
			}
		}
		return nil
	})
}

func replaceParenthesisRules(ri *RuleIndex) error {
	//replaceM := ri.m
	replaceM := map[string]Rule{}

	return visitRulesOnce(ri, func(rref *Rule) error {
		switch t := (*rref).(type) {
		case *ParenRule:
			//id := t.id()
			id := t.idSimple()
			r1, ok := replaceM[id]
			if ok {
				*rref = r1
				return nil // don't walk childs, already replaced
			}

			// replaced with a defrule with the translation to and/or rules
			dr := &DefRule{}
			dr.name = id
			*rref = dr
			replaceM[id] = dr

			switch t.typ {
			case parenrNone:
				dr.setOnlyChild(t.onlyChild())
			case parenrOptional:
				r3 := t.onlyChild()
				r4 := &OrRule{}
				r4.childs2 = []Rule{r3, nilRule}
				dr.setOnlyChild(r4)
			case parenrZeroOrMore:
				r2 := t.onlyChild()
				r3 := &AndRule{}
				r3.childs2 = []Rule{dr, r2} // loop
				r4 := &OrRule{}
				r4.childs2 = []Rule{r3, nilRule}
				dr.setOnlyChild(r4)
				dr.isLoop = true
			case parenrOneOrMore:
				r2 := t.onlyChild()
				r3 := &AndRule{}
				r3.childs2 = []Rule{dr, r2} // loop
				r4 := &OrRule{}
				//r4.childs = []Rule{r3, r2}

				// NOTE: provide a wrap to the element to have the loop detect and build a slice of elements (andrule with nil is harmless)
				r5 := &AndRule{}
				r5.childs2 = []Rule{r2, nilRule}
				r4.childs2 = []Rule{r3, r5} // last element

				dr.setOnlyChild(r4)
				dr.isLoop = true
			default:
				return goutil.TodoErrorStr(fmt.Sprintf("%q", t.typ))
			}
		}
		return nil
	})
}

//----------

// ex: parenthesis rules are replaced by an unique instance, that is, all instances of "(a|b)" will have a unique instance
func makeRulesUnique(ri *RuleIndex) error {
	unique := map[string]*Rule{}
	_ = visitRulesOnce(ri, func(rref *Rule) error {
		_ = replaceFromMap(unique, (*rref).id(), rref)
		return nil
	})
	return nil
}

//----------

func replaceFromMap(m map[string]*Rule, id string, rref *Rule) bool {
	r2, ok := m[id]
	if ok {
		// replace reference with the one already existent
		*rref = *r2
		return true
	}
	m[id] = rref // keep
	return false // not replaced
}

func visitRulesOnce(ri *RuleIndex, fn func(*Rule) error) error {
	seen := map[Rule]bool{}
	fn2 := (func(rref *Rule) error)(nil)
	fn2 = func(rref *Rule) error {
		if seen[*rref] {
			return nil
		}
		seen[*rref] = true

		//// walk childs first (TESTING: case of ifrule)
		//if err := walkRuleChilds(*rref, fn2); err != nil {
		//	return err
		//}

		// this node
		if err := fn(rref); err != nil {
			return err
		}
		// walk childs after
		return walkRuleChilds(*rref, fn2)
	}
	for _, r := range ri.m {
		if err := fn2(r); err != nil {
			return err
		}
	}
	return nil
}

//----------
//----------
//----------

func parseLetter(ps *PState) error {
	ps2 := ps.copy()
	ru, err := ps2.readRune()
	if err != nil {
		return err
	}
	if !unicode.IsLetter(ru) {
		return errors.New("not a letter")
	}
	ps.set(ps2)
	return nil
}
func parseDigit(ps *PState) error {
	ps2 := ps.copy()
	ru, err := ps2.readRune()
	if err != nil {
		return err
	}
	if !unicode.IsDigit(ru) {
		return errors.New("not a digit")
	}
	ps.set(ps2)
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
