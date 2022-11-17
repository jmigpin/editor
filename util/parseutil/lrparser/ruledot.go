package lrparser

import (
	"fmt"
	"sort"
	"strings"
)

type RuleDot struct { // also know as "item"
	prod    Rule // producer: "prod->rule"
	rule    Rule // producee: where the dot runs
	dot     int  // rule dot
	reverse bool
}

func newRuleDot(prod, rule Rule, reverse bool) *RuleDot {
	rd := &RuleDot{prod: prod, rule: rule}
	if reverse && ruleProdCanReverse(prod) {
		rd.reverse = true
		//rd.dot = len(rd.sequence())
	}
	rd.advanceDotNils()
	return rd
}

//----------

func (rd *RuleDot) sequence() []Rule {
	return ruleSequence(rd.rule, rd.reverse)
}

//----------

func (rd *RuleDot) dotRule() (Rule, bool) {
	if rd.dotAtEnd() {
		return nil, false
	}
	w := rd.sequence()
	//if rd.reverse {
	//	return w[rd.dot-1], true
	//}
	return w[rd.dot], true
}
func (rd *RuleDot) dotAtEnd() bool {
	//if rd.reverse {
	//	return rd.dot == 0
	//}
	return rd.dot == len(rd.sequence())
}
func (rd *RuleDot) dotAndAfterRules() []Rule {
	// assumes valid dot
	w := rd.sequence()
	//if rd.reverse {
	//	//return reverseRulesCopy(w[:rd.dot])
	//	return w[:rd.dot]
	//}
	return w[rd.dot:]
}

//----------

func (rd *RuleDot) advanceDot() (*RuleDot, bool) {
	if rd.dotAtEnd() {
		return nil, false
	}
	rd2 := *rd // copy
	rd2.blindlyAdvanceDot()
	rd2.advanceDotNils()
	return &rd2, true
}
func (rd *RuleDot) blindlyAdvanceDot() {
	//if rd.reverse {
	//	rd.dot--
	//} else {
	//	rd.dot++
	//}
	rd.dot++
}
func (rd *RuleDot) advanceDotNils() {
	for {
		r, ok := rd.dotRule()
		if ok && r == nilRule {
			rd.blindlyAdvanceDot()
			continue
		}
		break
	}
}

//----------

func (rd *RuleDot) popLen() int {
	w := rd.sequence()
	// don't count nils
	k := 0
	for _, r := range w {
		if r == nilRule {
			continue
		}
		k++
	}
	return k
}

//----------

func (rd *RuleDot) String() string {
	rev := ""
	if rd.reverse {
		rev = "rev:"
	}
	return fmt.Sprintf("{%v%v,%v->%v}", rev, rd.dot, rd.prod.id(), rd.rule.id())
}

//----------
//----------
//----------

type RuleDots []*RuleDot

func (rds RuleDots) has(rd *RuleDot) bool {
	for _, rd2 := range rds {
		if *rd2 == *rd {
			return true
		}
	}
	return false
}
func (rds RuleDots) dotRulesSet() RuleSet {
	rset := RuleSet{}
	for _, rd := range rds {
		if r, ok := rd.dotRule(); ok {
			rset.set(r)
		}
	}
	return rset
}
func (rds RuleDots) sorted() RuleDots {
	w := make(RuleDots, len(rds))
	copy(w, rds)
	sortRuleDots(w)
	return w
}
func (rds RuleDots) String() string {
	s := "ruledots:\n"
	rds = rds.sorted()
	for _, rd := range rds {
		s += fmt.Sprintf("\t%v\n", rd)
	}
	return strings.TrimSpace(s)
}

//----------
//----------
//----------

func sortRuleDots(w RuleDots) {
	sort.Slice(w, func(a, b int) bool {
		ra, rb := w[a], w[b]
		va1, va2, va3, va4, va5 := sortRuleDotsValue(ra)
		vb1, vb2, vb3, vb4, vb5 := sortRuleDotsValue(rb)
		if va1 == vb1 {
			if va2 == vb2 {
				if va3 == vb3 {
					if va4 == vb4 {
						return va5 < vb5
					}
					return va4 < vb4
				}
				return va3 < vb3
			}
			return va2 < vb2
		}
		return va1 < vb1
	})
}
func sortRuleDotsValue(rd *RuleDot) (int, string, int, string, int) {
	ta, sa := sortRulesValue(rd.prod)
	tb, sb := sortRulesValue(rd.rule)
	return ta, sa, tb, sb, rd.dot
}
