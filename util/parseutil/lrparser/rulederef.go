package lrparser

import (
	"fmt"
	"sort"

	"github.com/jmigpin/editor/util/goutil"
)

func dereferenceRules(ri *RuleIndex) error {
	// replace refrules to avoid rule ids with "refs", and catch first errors in case a refrule does not exist
	if err := replaceRefRules(ri); err != nil {
		return err
	}
	// checks boolrule value (now), can run only after replaceRefRules
	if err := replaceIfRules(ri); err != nil {
		return err
	}

	if err := replaceStringRuleRefs(ri); err != nil {
		return err
	}
	if err := replaceProcRules(ri); err != nil {
		return err
	}
	if err := replaceParenthesisRules(ri); err != nil {
		return err
	}
	if err := replaceDuplicateRules(ri); err != nil {
		return err
	}

	// sanity check: rules not allowed after deref phase
	return visitRuleIndexTree(ri, func(rref *Rule) error {
		switch t := (*rref).(type) {
		case *RefRule,
			*ParenRule,
			*IfRule,
			//*BoolRule, // commented: some residual rule not used in an "if" will still be present // TODO: make a clear step of boolrules?
			*ProcRule:
			err := fmt.Errorf("rule type present after deref phase: %T, %v", t, t)
			//return err
			panic(err)
		}
		return nil
	})

	return nil
}

//----------

func replaceRefRules(ri *RuleIndex) error {
	return visitRuleIndexTree(ri, func(rref *Rule) error {
		switch t := (*rref).(type) {
		case *RefRule:
			// replace with rule in ruleindex
			if !replaceFromMap(ri.m, t.name, rref) {
				err := fmt.Errorf("rule not found: %v", t.name)
				return &PosError{err: err, Pos: t.Pos()}
			}
		}
		return nil
	})
}
func replaceStringRuleRefs(ri *RuleIndex) error {
	// replace rest of ref rules (stringrule refs)
	return visitRuleIndexTree(ri, func(rref *Rule) error {
		switch t := (*rref).(type) {
		case *StringRule:
			if len(t.childs()) == 1 { // needs dereference
				r := t.childs()[0]
				sr, ok := ruleInnerStringRule(r, t.typ)
				if !ok {
					return nodePosErrorf(t, "expecting a compatible derivation of stringrules: %v", r)
				}
				t.childs_ = nil // dereferenced
				t.runes = sr.runes
			}
		}
		return nil
	})
}

func replaceProcRules(ri *RuleIndex) error {
	return visitRuleIndexTree(ri, func(rref *Rule) error {
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
	return visitRuleIndexTree(ri, func(rref *Rule) error {
		switch t := (*rref).(type) {
		case *IfRule:
			c0 := t.childs_[0] // conditional rule
			c1 := t.childs_[1] // rule if condition is true
			c2 := t.childs_[2] // rule if condition is false
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

// the rule index will not have parenthesis rules after this step, as they will be transformed into defrule with the equivalent id, using and/or rules
func replaceParenthesisRules(ri *RuleIndex) error {
	// parenthesis defrule name
	lzc := 0  // loop zero counter
	loc := 0  // loop one counter
	optc := 0 // optional counter
	pname := func(t parenrType) string {
		ts := ""
		switch t {
		case parenrOptional:
			ts = fmt.Sprintf("opt%d", optc)
			optc++
		case parenrZeroOrMore:
			ts = fmt.Sprintf("lz%d", lzc)
			lzc++
		case parenrOneOrMore:
			ts = fmt.Sprintf("lo%d", loc)
			loc++
		default:
			panic("!")
		}
		return fmt.Sprintf("%s", ts)
	}
	_ = pname

	unique := map[string]*DefRule{}
	newDefRule := func(pr *ParenRule) *DefRule {
		id := pr.id()
		dr, ok := unique[id]
		if ok {
			return dr
		}
		//dr = &DefRule{name: pname(pr.typ)}
		dr = &DefRule{name: id}
		unique[id] = dr
		if err := ri.set(dr.name, dr); err != nil {
			panic(err)
		}
		return dr
	}

	visit := (visitRuleRefFn)(nil)
	visit = wrapVisitChilds(func(rref *Rule) error {
		switch t := (*rref).(type) {
		case *ParenRule:
			// replace with defrule with special name
			switch t.typ {
			case parenrNone:
				*rref = t.onlyChild()
				//return visit(rref) // visit the new rref itself
			case parenrOptional:
				dr := newDefRule(t)
				r2 := t.onlyChild()
				r4 := &OrRule{}
				r4.childs_ = []Rule{r2, nilRule}
				dr.setOnlyChild(r4)
				dr.isPOptional = true
				*rref = dr
			case parenrZeroOrMore:
				dr := newDefRule(t)
				r2 := t.onlyChild()
				r3 := &AndRule{}
				r3.childs_ = []Rule{dr, r2} // loop before (smaller run stack // also allows less conflicts due to left-to-right?) // order also used in node.go childloop func
				//r3.childs2 = []Rule{r2, dr} // loop after
				r4 := &OrRule{}
				r4.childs_ = []Rule{r3, nilRule}
				dr.setOnlyChild(r4)
				dr.isNoReverse = true
				dr.isPZeroOrMore = true
				*rref = dr
			case parenrOneOrMore:
				//// own loop
				//// - has issues with early stop because there is no nil rule to recover with
				//dr := newDefRule(t)
				//r2 := t.onlyChild()
				//r3 := &AndRule{}
				//r3.childs2 = []Rule{dr, r2} // loop before (smaller run stack)
				////r3.childs2 = []Rule{r2, dr} // loop after
				//r4 := &OrRule{}
				//r4.childs2 = []Rule{r3, r2}
				//dr.setOnlyChild(r4)
				//dr.isNoReverse = true
				//*rref = dr

				// with zeroormore
				dr := newDefRule(t)
				r2 := t.onlyChild()
				r3 := &ParenRule{}
				r3.typ = parenrZeroOrMore
				r3.setOnlyChild(r2)
				r4 := &AndRule{}
				r4.childs_ = []Rule{r3, r2} // place loop before // order also used in node.go childloop func
				//r4.childs2 = []Rule{r2, r3} // place loop after // TODO: fails testlrparser21
				dr.setOnlyChild(r4)
				dr.isNoReverse = true
				dr.isPOneOrMore = true
				*rref = dr
			default:
				return goutil.TodoErrorStr(fmt.Sprintf("%q", t.typ))
			}

			// visit the new rref itself
			return visit(rref)
		}
		return nil
	})
	return visitRuleIndexRules(ri, visit)
}

// make rules unique
// - the pos is lost since the repeated rules are replaced with the first definition
// - the rule src position must not be used after this function
func replaceDuplicateRules(ri *RuleIndex) error {
	unique := map[string]*Rule{}
	return visitRuleIndexTree(ri, func(rref *Rule) error {
		_ = replaceFromMap(unique, (*rref).id(), rref)
		return nil
	})
}

//----------
//----------
//----------

func visitRuleIndexTree(ri *RuleIndex, fn visitRuleRefFn) error {
	visit := (visitRuleRefFn)(nil) // example on how fn could refer to visit inside
	visit = wrapVisitChilds(fn)
	return visitRuleIndexRules(ri, visit)
}

func visitRuleIndexRules(ri *RuleIndex, fn visitRuleRefFn) error {
	//for _, r := range ri.m {
	//	r := ri.m[k]
	//	if err := fn(r); err != nil {
	//		return err
	//	}
	//}

	// stable iteration to avoid (if used) unstable parenthesis loop names
	ks := []string{}
	for k := range ri.m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	for _, k := range ks {
		r := ri.m[k]
		if err := fn(r); err != nil {
			return err
		}
	}
	return nil
}

func wrapVisitChilds(fn visitRuleRefFn) visitRuleRefFn {
	seen := map[Rule]bool{} // avoid loops
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
	return fn2

	//fns := wrapVisitSeen(fn)
	//fn2 := (func(rref *Rule) error)(nil)
	//fn2 = func(rref *Rule) error {
	//	if err := fns(rref); err != nil {
	//		return err
	//	}
	//	return walkRuleChilds(*rref, fn2)
	//}
	//return fn2
}

//func wrapVisitSeen(fn visitRuleRefFn) visitRuleRefFn {
//	seen := map[Rule]bool{}
//	fn2 := (func(rref *Rule) error)(nil)
//	fn2 = func(rref *Rule) error {
//		if seen[*rref] {
//			return nil
//		}
//		seen[*rref] = true
//		return fn(rref)
//	}
//	return fn2
//}

type visitRuleRefFn func(*Rule) error

//----------
//----------
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

//----------

func nodePosErrorf(n PNode, f string, args ...interface{}) error {
	err := fmt.Errorf(f, args...)
	return &PosError{err: err, Pos: n.Pos()}
}
