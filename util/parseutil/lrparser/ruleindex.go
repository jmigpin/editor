package lrparser

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// unique rule index
type RuleIndex struct {
	m map[string]Rule // *DefRule,*FuncRule

	deref struct {
		once bool
		err  error
	}
}

func newRuleIndex() *RuleIndex {
	ri := &RuleIndex{m: map[string]Rule{}}

	setSingleton := func(r *SingletonRule) {
		ri.m[r.name] = r // ok to set directly, only for special rules
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

	// need a level on indirection to have the ruleindex.map be iterable withotu issues when making rules unique. Forcing all rules in the index to be either defrule or funcrule provides that, while allowing directly a stringrule would cause issues
	switch r.(type) {
	case *DefRule, *FuncRule:
	default:
		return fmt.Errorf("unexpected type to set in ruleindex: %T", r)
	}

	switch name {
	case "", "rule",
		endRule.id(),
		nilRule.id(),
		anyruneRule.id():
		return fmt.Errorf("bad rule name: %q", name)
	}
	if ri.has(name) {
		return fmt.Errorf("rule already set: %v", name)
	}

	ri.m[name] = r
	return nil
}
func (ri *RuleIndex) has(name string) bool {
	_, ok := ri.m[name]
	return ok
}
func (ri *RuleIndex) get(name string) (Rule, bool) {
	r, ok := ri.m[name]
	return r, ok
}
func (ri *RuleIndex) delete(name string) {
	delete(ri.m, name)
}

//----------

func (ri *RuleIndex) setFuncRule(name string, fn pstateParseFn) error {
	fr := &FuncRule{name: name, fn: fn}
	return ri.set(name, fr)
}
func (ri *RuleIndex) setDefRule(name string, r Rule) error {
	dr := &DefRule{name: name}
	dr.setOnlyChild(r)
	return ri.set(name, dr)
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
		dr, ok := r.(*DefRule)
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
	return "ruleindex:\n" + strings.Join(res, "\n")
}

func (ri *RuleIndex) sorted() []Rule {
	w := []Rule{}
	for _, r := range ri.m {
		w = append(w, r)
	}
	sortRules(w)
	return w
}

//----------
//----------
//----------

func dereferenceRules(ri *RuleIndex) error {
	// replace refrules first to avoid rule ids with "refs"
	if err := derefRefRules(ri); err != nil {
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

func derefRefRules(ri *RuleIndex) error {
	return visitRulesOnce(ri, func(rref *Rule) error {
		switch t := (*rref).(type) {
		case *RefRule:
			// replace with defrule in ruleindex
			if !replaceFromMap(ri.m, t.name, rref) {
				err := fmt.Errorf("rule not found: %v", t.name)
				return &PosError{err: err, Pos: t.pos}
			}
		}
		return nil
	})
}

func replaceParenthesisRules(ri *RuleIndex) error {
	//replaceM := ri.m
	replaceM := map[string]Rule{}

	replace := func(tag string, id string, rref *Rule) (*DefRule, bool) {
		r1, ok := replaceM[id]
		if ok {
			*rref = r1
			return nil, true // replaced
		}
		// create
		dr := &DefRule{}
		dr.name = id
		//dr.name =fmt.Sprintf("{%v:%v}",tag,id) // DEBUG
		dr.setOnlyChild(*rref)
		*rref = dr
		replaceM[id] = dr
		return dr, false // not replaced (created)
	}

	return visitRulesOnce(ri, func(rref *Rule) error {
		switch t := (*rref).(type) {
		case *ParenRule:
			dr, replaced := replace("par", t.id(), rref)
			if replaced {
				return nil // don't walk childs, already replaced
			}
			dr.setOnlyChild(t.onlyChild())

		case *ParenOptionalRule:
			dr, replaced := replace("pOpt", t.id(), rref)
			if replaced {
				return nil
			}
			r3 := &ParenRule{}
			r3.setOnlyChild(t.onlyChild())
			r4 := &OrRule{}
			r4.childs = []Rule{r3, nilRule}
			dr.setOnlyChild(r4)

		case *ParenZeroOrMoreRule:
			dr, replaced := replace("pZom", t.id(), rref)
			if replaced {
				return nil
			}
			r2 := &ParenRule{}
			r2.setOnlyChild(t.onlyChild())
			r3 := &AndRule{}
			r3.childs = []Rule{dr, r2} // loop
			r4 := &OrRule{}
			r4.childs = []Rule{r3, nilRule} // last element
			dr.setOnlyChild(r4)
			dr.isLoop = true

		case *ParenOneOrMoreRule:
			dr, replaced := replace("pOom", t.id(), rref)
			if replaced {
				return nil
			}
			r2 := &ParenRule{}
			r2.setOnlyChild(t.onlyChild())
			r3 := &AndRule{}
			r3.childs = []Rule{dr, r2} // loop
			r4 := &OrRule{}
			r4.childs = []Rule{r3, r2} // last element
			dr.setOnlyChild(r4)
			dr.isLoop = true
		}
		return nil
	})
}

//----------

// ex: parenthesis rules are replaced by an unique instance, that is, all instances of "(a|b)" will have a unique instance
func makeRulesUnique(ri *RuleIndex) error {
	unique := map[string]Rule{}
	_ = visitRulesOnce(ri, func(rref *Rule) error {
		_ = replaceFromMap(unique, (*rref).id(), rref)
		return nil
	})
	return nil
}

//----------

func replaceFromMap(m map[string]Rule, id string, rref *Rule) bool {
	r2, ok := m[id]
	if ok {
		// replace reference with the one already existent
		*rref = r2
		return true
	}
	m[id] = *rref // keep
	return false  // not replaced
}

func visitRulesOnce(ri *RuleIndex, fn func(*Rule) error) error {
	seen := map[Rule]bool{}
	fn2 := (func(rref *Rule) error)(nil)
	fn2 = func(rref *Rule) error {
		if seen[*rref] {
			return nil
		}
		seen[*rref] = true
		if err := fn(rref); err != nil {
			return err
		}
		return walkRuleChilds(*rref, fn2)
	}

	for _, r := range ri.m {
		if err := fn2(&r); err != nil { // NOTE: r is local
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
