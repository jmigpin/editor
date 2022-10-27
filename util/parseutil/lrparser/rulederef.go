package lrparser

import (
	"fmt"

	"github.com/jmigpin/editor/util/goutil"
)

func dereferenceRules(ri *RuleIndex) error {
	// replace refrules first to avoid rule ids with "refs"
	if err := replaceRefRules(ri); err != nil {
		return err
	}

	if err := replaceIfRules(ri); err != nil {
		return err
	}
	if err := replaceRefRules2(ri); err != nil {
		return err
	}
	if err := replaceProcRules(ri); err != nil {
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
	// replace ref rule (non stringrule refs, better first errors)
	return visitRulesOnce(ri, func(rref *Rule) error {
		switch t := (*rref).(type) {
		case *RefRule:
			rref2 := rref
			if t.srRef.typ != stringrNone { // don't replace yet if it has a stringrType, just keep for later
				rref2 = &t.srRef.r
			}
			// replace with rule in ruleindex
			if !replaceFromMap(ri.m, t.name, rref2) {
				err := fmt.Errorf("rule not found: %v", t.name)
				return &PosError{err: err, Pos: t.Pos()}
			}
		}
		return nil
	})
}
func replaceRefRules2(ri *RuleIndex) error {
	// replace rest of ref rules (stringrule refs)
	return visitRulesOnce(ri, func(rref *Rule) error {
		switch t := (*rref).(type) {
		case *RefRule:
			sr, ok := ruleInnerStringRule(t.srRef.r, t.srRef.typ)
			if !ok {
				return nodePosErrorf(t, "expecting a compatible derivation of stringrules")
			}
			sr2 := *sr // copy (to set type)
			sr2.typ = t.srRef.typ
			*rref = &sr2
		}
		return nil
	})
}

func replaceProcRules(ri *RuleIndex) error {
	return visitRulesOnce(ri, func(rref *Rule) error {
		switch t := (*rref).(type) {
		case *ProcRule:
			fn, ok := ri.cm[t.name]
			if !ok {
				return nodePosErrorf(t, "call rule not found: %v", t.name)
			}
			if u, err := fn(t.onlyChild()); err != nil {
				return nodePosErrorf(t, "%v: %w", t.name, err)
			} else {
				*rref = u
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
				//r4.childs2 = []Rule{r3, r2}

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

func nodePosErrorf(n PNode, f string, args ...interface{}) error {
	err := fmt.Errorf(f, args...)
	return &PosError{err: err, Pos: n.Pos()}
}
