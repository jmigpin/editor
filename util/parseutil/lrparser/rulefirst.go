package lrparser

import (
	"fmt"
	"strings"
)

// rules first terminals
type RuleFirstT struct {
	ri      *RuleIndex
	cache   map[Rule]RuleSet
	seen    map[Rule]int
	reverse bool
}

func newRuleFirstT(ri *RuleIndex, reverse bool) *RuleFirstT {
	rf := &RuleFirstT{ri: ri, reverse: reverse}
	rf.cache = map[Rule]RuleSet{}
	rf.seen = map[Rule]int{}
	return rf
}

//----------

func (rf *RuleFirstT) first(r Rule) RuleSet {
	rset, ok := rf.cache[r]
	if ok {
		return rset
	}

	rf.seen[r]++
	defer func() { rf.seen[r]-- }()
	if rf.seen[r] > 2 { // extra loop to allow proper solve
		return nil
	}

	rset = RuleSet{}
	if r.isTerminal() {
		rset.set(r)
	} else {
		inReverse := rf.reverse && ruleProdCanReverse(r)
		for _, r2 := range ruleProductions(r) { // r->a0|...|an
			w2 := ruleSequence(r2, inReverse) // r->a0 ... an
			rset2 := rf.sequenceFirst(w2)
			rset.add(rset2)
		}
	}
	rf.cache[r] = rset
	return rset
}
func (rf *RuleFirstT) sequenceFirst(w []Rule) RuleSet {
	rset := RuleSet{}
	allHaveNil := true
	for _, r := range w { // w -> r1 ... rk
		rset2 := rf.first(r)
		rset.add(rset2)
		if !rset2.has(nilRule) {
			allHaveNil = false
			break
		}
	}
	if !allHaveNil {
		rset.unset(nilRule)
	}
	return rset
}

//----------

func (rf *RuleFirstT) String() string {
	u := []string{}
	for _, r := range rf.ri.sorted() {
		if r.isTerminal() { // no need to show terminals
			continue
		}
		u = append(u, fmt.Sprintf("%v:%v", r.id(), rf.first(r)))
	}
	return fmt.Sprintf("rulefirst[rev=%v]{\n\t%v\n}", rf.reverse, strings.Join(u, "\n\t"))
}

//----------
//----------
//----------

//type RuleFollow struct {
//	ri     *RuleIndex
//	rFirst *RulesFirst
//	cache  map[Rule]RuleSet
//}

//func newRuleFollow(ri *RuleIndex, rFirst *RulesFirst, r Rule) *RuleFollow {
//	rf := &RuleFollow{ri: ri, rFirst: rFirst}
//	rf.cache = map[Rule]RuleSet{}
//	rf.calc(r)
//	return rf
//}
//func (rf *RuleFollow) get(r Rule) RuleSet {
//	return rf.cache[r]
//}
//func (rf *RuleFollow) calc(r Rule) {
//	AFollow := RuleSet{}
//	AFollow.set(rf.ri.endRule())
//	rf.cache[r] = AFollow

//	seen := map[Rule]int{}
//	rf.calc2(r, AFollow, seen)
//}
//func (rf *RuleFollow) calc2(A Rule, AFollow RuleSet, seen map[Rule]int) {
//	if seen[A] >= 2 { // need to visit 2nd time to allow afollow to be used in nested rules
//		return
//	}
//	seen[A]++
//	defer func() { seen[A]-- }()

//	//rset := RuleSet{}
//	w, ok := ruleProductions(A)
//	if !ok { // terminal
//		return
//	}
//	nilr := rf.ri.nilRule()
//	for _, r2 := range w {
//		// A->r2
//		w2 := ruleRhs(r2) // sequence
//		for i, B := range w2 {
//			// A->αBβ

//			if ruleIsTerminal(B) {
//				continue
//			}

//			BFollow, ok := rf.cache[B]
//			if !ok {
//				BFollow = RuleSet{}
//				rf.cache[B] = BFollow
//			}

//			haveβ := i < len(w2)-1
//			βFirstHasNil := false
//			if haveβ {
//				β := w2[i+1]
//				βFirst := RuleSet{}
//				βFirst.add(rf.rFirst.get(β))
//				βFirstHasNil = βFirst.isSet(nilr)
//				βFirst.unset(nilr)
//				BFollow.add(βFirst)
//			}
//			if !haveβ || βFirstHasNil {
//				BFollow.add(AFollow)
//			}

//			rf.calc2(B, BFollow, seen)
//		}
//	}
//}
//func (rf *RuleFollow) String() string {
//	u := []string{}
//	for _, r := range rf.ri.sorted() {
//		u = append(u, fmt.Sprintf("%v:%v", r.id(), rf.get(r)))
//	}
//	return fmt.Sprintf("{\n\t%v\n}", strings.Join(u, ",\n\t"))
//}
