package lrparser

import (
	"fmt"
)

// aka "set of items"
type RuleDotsLaSet map[RuleDot]RuleSet // lookahead set

func (rdslas RuleDotsLaSet) setRuleSet(rd RuleDot, rset RuleSet) {
	if _, ok := rdslas[rd]; !ok {
		rdslas[rd] = RuleSet{}
	}
	rdslas[rd].add(rset)
}
func (rdslas RuleDotsLaSet) setRule(rd RuleDot, r Rule) {
	if _, ok := rdslas[rd]; !ok {
		rdslas[rd] = RuleSet{}
	}
	rdslas[rd].set(r)
}
func (rdslas RuleDotsLaSet) hasRule(rd RuleDot, r Rule) bool {
	rset, ok := rdslas[rd]
	if !ok {
		return false
	}
	return rset.has(r)
}

//----------

func (rdslas RuleDotsLaSet) ruleDots() RuleDots {
	w := RuleDots{}
	for rd := range rdslas {
		u := rd
		w = append(w, &u)
	}
	return w
}

//----------

func (rdslas RuleDotsLaSet) String() string {
	rds := RuleDots{}
	for k := range rdslas {
		u := k
		rds = append(rds, &u)
	}
	rds = rds.sorted()
	s := ""
	for _, rd := range rds {
		rset := rdslas[*rd]
		u := fmt.Sprintf("[%v,%v]", rd, rset)
		s += fmt.Sprintf("%v\n", u)
	}
	return s
}

//----------
//----------
//----------

//// aka "set of items"
//type RuleDotLas []*RuleDotLa

//func (rdlas RuleDotLas) has(rdla *RuleDotLa) bool {
//	str := rdla.String()
//	for _, rdla2 := range rdlas {
//		if rdla2.String() == str {
//			return true
//		}
//	}
//	return false
//}
//func (rdlas RuleDotLas) hasAll(rdlas2 RuleDotLas) bool {
//	for _, rdla2 := range rdlas2 {
//		if rdlas.has(rdla2) {
//			return true
//		}
//	}
//	return false
//}
//func (rdlas RuleDotLas) appendUnique(rdlas2 RuleDotLas) RuleDotLas {
//	m := map[string]bool{}
//	for _, rdla := range rdlas {
//		id := rdla.String()
//		m[id] = true
//	}
//	for _, rdla := range rdlas2 {
//		id := rdla.String()
//		if !m[id] {
//			rdlas = append(rdlas, rdla)
//		}
//	}
//	return rdlas
//}
//func (rdlas RuleDotLas) String() string {
//	s := "ruledotlookaheads:\n"
//	for _, rdl := range rdlas {
//		s += fmt.Sprintf("\t%v\n", rdl)
//	}
//	return s
//}

//----------
//----------
//----------

//func rdlaLookahead(rdla *RuleDotLa, rFirst *RulesFirst) {
//	if rdla.parent == nil { // rdla is the start rule
//		rdla.looka.set(rFirst.ri.endRule()) // lookahead is the end rule
//		return
//	}

//	b := []Rule{}
//	rd, ok := rdla.parent.rd.advanceDot()
//	if ok {
//		w := rd.dotAndAfterRules()
//		b = append(b, w...)
//		//if ok {
//		//b = append(b, r)
//		//rdla.looka.add(rFirst.get(r))
//		//}
//	}

//	for _, a := range rdla.parent.looka.sorted() {
//		ha := append(b, a)
//		rset := rFirst.getSequence(ha)
//		rdla.looka.add(rset)
//	}

//	//nilr := rFirst.ri.nilRule()
//	//if len(rdla.looka) == 0 || rdla.looka.isSet(nilr) {
//	//	rdla.looka.add(rdla.parent.looka)
//	//}
//}
