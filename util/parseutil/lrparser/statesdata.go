package lrparser

import (
	"fmt"
	"strings"

	"github.com/jmigpin/editor/util/iout"
)

type StatesData struct {
	states            []*State
	shiftOnSRConflict bool
}

func newStatesData(vd *VerticesData, shiftOnSRConflict bool) (*StatesData, error) {
	sd := &StatesData{shiftOnSRConflict: shiftOnSRConflict}
	if err := sd.build(vd); err != nil {
		return nil, err
	}
	if err := sd.checkStringsConflicts(vd); err != nil {
		return sd, err // also return sd for debug
	}
	if err := sd.checkActionConflicts(vd); err != nil {
		return sd, err // also return sd for debug
	}
	return sd, nil
}

func (sd *StatesData) build(vd *VerticesData) error {
	// map all states (using ints)
	sd.states = make([]*State, len(vd.verts))
	for _, v := range vd.verts {
		id := stateId(v.id)
		sd.states[id] = newState(id)
	}

	addAction := func(st *State, r Rule, a Action) {
		st.action[r] = append(st.action[r], a)
	}

	// construct states
	for _, v := range vd.verts {
		st := sd.states[int(v.id)]

		// action: shift
		for r, v2 := range v.gotoVert {
			st2 := sd.states[int(v2.id)]
			if r.isTerminal() {
				a := &ActionShift{st: st2}
				addAction(st, r, a)
			} else {
				// goto transitions
				st.gotoSt[r] = st2
			}
		}
		// action: reduce
		for rd, las := range v.rdslasC {
			if !rd.dotAtEnd() {
				continue
			}
			if rd.prod == startRule {
				if las.has(endRule) {
					addAction(st, endRule, &ActionAccept{})
				}
			} else {
				for r2 := range las {
					a := &ActionReduce{
						prod: rd.prod,
						popN: rd.popLen(),
					}
					addAction(st, r2, a)
				}
			}
		}

		// commented: done above at "action shift"
		//// goto transitions
		//for r, v2 := range v.gotoVert {
		//	if !ruleIsTerminal(r) {
		//		st.gotoSt[r] = sd.states[int(v2.id)]
		//	}
		//}

		rset := RuleSet{}
		// compute rset for parsenextrule
		for rd := range v.rdslasK {
			if r, ok := rd.dotRule(); ok {
				//st.rset.add(vd.rFirst.first(r))
				rset.add(vd.rFirst.first(r))
			}
		}
		// compute lookahead rset for parsenextrule
		for rd, las := range v.rdslasC {
			if rd.dotAtEnd() {
				//st.rsetLa.add(las)
				rset.add(las)
			}
		}

		// remove nil rules from the rset to parse
		rset.unset(nilRule)

		st.rsetSorted = sortRuleSetForParse(rset)
		st.rsetHasEndRule = rset.has(endRule)
	}

	return nil
}

//----------

func (sd *StatesData) checkActionConflicts(vd *VerticesData) error {
	me := iout.MultiError{}
	for _, st := range sd.states {
		for r, as := range st.action {
			if len(as) <= 1 {
				continue
			}

			// solve shift/reduct conflicts with shift (ignores reductions in this action)
			if sd.shiftOnSRConflict {
				shifts := []Action{}
				for _, a := range as {
					if u, ok := a.(*ActionShift); ok {
						shifts = append(shifts, u)
					}
				}
				// prefer shift (don't reduce)
				if len(shifts) == 1 {
					st.action[r] = shifts
					continue
				}
			}

			// have conflict
			w := []string{}
			w = append(w, fmt.Sprintf("conflict: %v, %v:", st.id, r.id()))
			for _, a := range as {
				w = append(w, fmt.Sprintf("%v", a))
			}
			//v := vd.verts[st.id]
			//w = append(w, fmt.Sprintf("%v\n", v))
			//w = append(w, fmt.Sprintf("%v", st))
			err := fmt.Errorf("%v", strings.Join(w, "\n"))
			me.Add(err)
		}
	}
	return me.Result()
}

//----------

func (sd *StatesData) checkStringsConflicts(vd *VerticesData) error {
	// TODO: anyrune?

	for _, st := range sd.states {
		for i, r := range st.rsetSorted {
			sr1, ok := r.(*StringRule)
			if !ok {
				continue
			}

			for k := i + 1; k < len(st.rsetSorted); k++ {
				r2 := st.rsetSorted[k]
				sr2, ok := r2.(*StringRule)
				if !ok {
					continue
				}

				if ok, err := sr1.intersect(sr2); err == nil && ok {
					return fmt.Errorf("stringrules %v intersects with %v", sr2, sr1)
				}
			}
		}
	}
	return nil
}

//func (sd *StatesData) checkStringsConflicts2(sr1, sr2 *StringRule) error {
//	switch sr2.typ {
//	case stringRTOr:
//		for _, ru2 := range sr2.runes {
//			if has, err := sd.srHasRune(sr1, ru2); err != nil {
//				return err
//			} else if has {
//				return fmt.Errorf("rune %q already in %v", ru2, sr1)
//			}
//		}
//		for _, rr2 := range sr2.ranges {
//			for _, ru2 := range []rune{rr2[0], rr2[1]} {
//				if has, err := sd.srHasRune(sr1, ru2); err != nil {
//					return err
//				} else if has {
//					return fmt.Errorf("range %v already in %v", rr2, sr1)
//				}
//			}
//		}
//		//case stringRTOr:
//	}
//	return nil
//}

//func (sd *StatesData) srHasRune(sr *StringRule, ru rune) (bool, error) {
//	switch sr.typ {
//	case stringRTOr:
//		for _, ru2 := range sr.runes {
//			if ru == ru2 {
//				return true, nil
//			}
//		}
//		for _, rr := range sr.ranges {
//			if rr.HasRune(ru) {
//				return true, nil
//			}
//		}
//		return false, nil
//	case stringRTOrNeg:
//		for _, ru2 := range sr.runes {
//			if ru == ru2 {
//				return false, nil
//			}
//		}
//		for _, rr := range sr.ranges {
//			if !rr.HasRune(ru) {
//				return false, nil
//			}
//		}
//		return true, nil
//	}
//	return false, fmt.Errorf("not orrule")
//}

//func (sd *StatesData) srHasRune(sr *StringRule, ru rune) (bool, error) {
//	switch sr.typ {
//	case stringRTOr:
//		for _, ru2 := range sr.runes {
//			if ru == ru2 {
//				return true, nil
//			}
//		}
//		for _, rr := range sr.ranges {
//			if rr.HasRune(ru) {
//				return true, nil
//			}
//		}
//		return false, nil
//	case stringRTOrNeg:
//		for _, ru2 := range sr.runes {
//			if ru == ru2 {
//				return false, nil
//			}
//		}
//		for _, rr := range sr.ranges {
//			if !rr.HasRune(ru) {
//				return false, nil
//			}
//		}
//		return true, nil
//	}
//	return false, fmt.Errorf("not orrule")
//}

//func (sd *StatesData) runeConflict(sr *StringRule, ru rune) error {
//	switch sr.typ {
//	case stringRTOr:
//		for _, ru2 := range sr.runes {
//			if ru2 == ru {
//				return fmt.Errorf("rune %q already defined at %v", ru sr)
//			}
//		}
//		for _, rr := range sr.ranges {
//			if rr.HasRune(ru) {
//				return fmt.Errorf("rune %q already defined at %v", ru sr)
//			}
//		}
//		return false, nil
//	case stringRTOrNeg:
//		for _, ru2 := range sr.runes {
//			if ru == ru2 {
//				return false, nil
//			}
//		}
//		for _, rr := range sr.ranges {
//			if !rr.HasRune(ru) {
//				return false, nil
//			}
//		}
//		return true, nil
//	default:
//		panic(fmt.Sprintf("bad stringrule type: %q", sr.typ))
//	}
//}

//func (sd *StatesData) solveConflicts(vd *VerticesData) error {
//	// strings conflicts (runes)
//	for _, st := range sd.states {
//		orM := map[rune]Rule{}
//		orNegM := map[rune]Rule{}
//		orRangeM := map[RuneRange]Rule{}
//		orRangeNegM := map[RuneRange]Rule{}

//		hasAnyrune := false
//		for _, r := range st.rsetSorted {
//			if r == anyruneRule {
//				hasAnyrune = true
//				break
//			}
//		}

//		// check duplicates in orRules
//		for _, r := range st.rsetSorted {
//			sr, ok := r.(*StringRule)
//			if !ok {
//				continue
//			}

//			typ := sr.typ

//			// special case: check andRule as orRule
//			if typ == stringRTAnd && len(sr.runes) == 1 {
//				typ = stringRTOr
//			}

//			switch typ {
//			//case stringRTAnd: // sequence
//			//case stringRTMid: // sequence
//			case stringRTOr:
//				if err := sd.checkRuneDups(orM, st, sr, sr.runes...); err != nil {
//					return err
//				}
//				if err := sd.checkRangeDups(orRangeM, st, sr, sr.ranges...); err != nil {
//					return err
//				}
//			case stringRTOrNeg:
//				if err := sd.checkRuneDups(orNegM, st, sr, sr.runes...); err != nil {
//					return err
//				}
//				if err := sd.checkRangeDups(orRangeNegM, st, sr, sr.ranges...); err != nil {
//					return err
//				}
//			}
//		}

//		// check intersections: between individual runes and ranges
//		if err := sd.checkRunesRangesDups(orM, orRangeM, st); err != nil {
//			return err
//		}
//		if err := sd.checkRunesRangesDups(orNegM, orRangeNegM, st); err != nil {
//			return err
//		}

//		// check intersections: all "or" rules must be in "negation" if it is defined (ex: (a|b|(c|a|b)!)
//		if err := sd.checkRunesNegation(orM, orNegM, orRangeNegM, st); err != nil {
//			return err
//		}
//		//if err := sd.checkRangesNegation(orM, orNegM, st); err != nil {
//		//	return err
//		//}

//		// check conflicts: all "or" runes must be in "not"
//		if len(orNegM) > 0 {
//			for ru, r := range orM {
//				_, ok := orNegM[ru]
//				if !ok {
//					// show "not" rules
//					rs := &RuleSet{}
//					for _, r2 := range orNegM {
//						rs.set(r2)
//					}

//					return fmt.Errorf("%v: rune %q in %v is covered in %v", st.id, ru, r, rs)
//				}
//			}
//		}
//		if hasAnyrune {
//			if len(orM) > 0 || len(orNegM) > 0 {
//				return fmt.Errorf("%v: anyrune and stringrule in the same state\n%v", st.id, sd)
//			}
//		}
//	}

//}
//func (sd *StatesData) checkRuneDups(m map[rune]Rule, st *State, r Rule, rs ...rune) error {
//	for _, ru := range rs {
//		r2, ok := m[ru]
//		if ok {
//			return fmt.Errorf("%v: rune %q in %v is already defined at %v", st.id, ru, r, r2)
//		}
//		m[ru] = r
//	}
//	return nil
//}
//func (sd *StatesData) checkRangeDups(m map[RuneRange]Rule, st *State, r Rule, h ...RuneRange) error {
//	for _, rr := range h {
//		for rr2, r2 := range m {
//			if rr2.IntersectsRange(rr) {
//				return fmt.Errorf("%v: range %q in %v is already defined at %v", st.id, rr, r, r2)
//			}
//		}
//		m[rr] = r
//	}
//	return nil
//}
//func (sd *StatesData) checkRunesRangesDups(m1 map[Rune]Rule, m2 map[RuneRange]Rule, st *State) error {
//	for ru, r1 := range m1 {
//		for rr, r2 := range m2 {
//			if rr.HasRune(ru) {
//				return fmt.Errorf("%v: rune %q in %v is covered by range %v", st.id, ru, r1, rr)
//			}
//		}
//		m[rr] = r
//	}
//	return nil
//}
//func (sd *StatesData) checkRunesNegation(m, neg map[Rune]Rule, negRange map[RuneRange]Rule, st *State) error {
//	// all "or" runes must be in "neg"
//	if len(neg) > 0 {
//		for ru, r := range m {
//			_, ok := neg[ru]
//			if ok {
//				continue
//			}
//			// show "not" rules
//			rs := &RuleSet{}
//			for _, r2 := range neg {
//				rs.set(r2)
//			}
//			return fmt.Errorf("%v: rune %q in %v is covered in %v", st.id, ru, r, rs)
//		}
//	}
//	return nil
//}
//func (sd *StatesData) checkRunesNegation2(m map[Rune]Rule, neg map[RuneRange]Rule, st *State) error {
//	if len(neg) == 0 {
//		return nil
//	}
//	// all "or" runes must be in "neg"
//	for ru, r := range m {
//		for rr,r2:=range neg{
//			if rr.HasRune(ru)[

//			}
//		}
//		_, ok := neg[ru]
//		if ok {
//			continue
//		}
//		// show "not" rules
//		rs := &RuleSet{}
//		for _, r2 := range neg {
//			rs.set(r2)
//		}
//		return fmt.Errorf("%v: rune %q in %v is covered in %v", st.id, ru, r, rs)
//	}
//	return nil
//}

//----------

//godebug:annotateoff
func (sd *StatesData) String() string {
	sb := &strings.Builder{}
	for _, st := range sd.states {
		fmt.Fprintf(sb, "%v\n", st)
	}
	return sb.String()
}

//----------
//----------
//----------

type State struct {
	id             stateId
	action         map[Rule][]Action
	gotoSt         map[Rule]*State
	rsetSorted     []Rule // rule set to parse in this state
	rsetHasEndRule bool
}

func newState(id stateId) *State {
	st := &State{id: id}
	st.action = map[Rule][]Action{}
	st.gotoSt = map[Rule]*State{}
	return st
}

func (st *State) actionRulesSorted() []Rule {
	w := []Rule{}
	for r := range st.action {
		w = append(w, r)
	}
	sortRules(w)
	return w
}

//godebug:annotateoff
func (st *State) String() string {
	s := fmt.Sprintf("%v:\n", st.id)

	s += "\tactions:\n"
	for _, r := range st.actionRulesSorted() {
		a := st.action[r]
		//u := fmt.Sprintf("%v(%p,%T)-> %v\n", r.id(), r, r, a)
		u := fmt.Sprintf("%v -> %v\n", r.id(), a)
		s += indentStr("\t\t", u)
	}

	s += "\tgotos:\n"
	for r, st2 := range st.gotoSt {
		u := fmt.Sprintf("%v -> %v\n", r.id(), st2.id)
		s += indentStr("\t\t", u)
	}

	//s += indentStr("\t", "rset: "+st.rset.String())
	//s += indentStr("\t", "la rset: "+st.rsetLa.String())
	s += indentStr("\t", fmt.Sprintf("rset: %v", st.rsetSorted))
	s = strings.TrimSpace(s)
	return s
}

//----------
//----------
//----------

type stateId int

func (sid stateId) String() string {
	return fmt.Sprintf("state%d", int(sid))
}

//----------
//----------
//----------

type Action interface{}

type ActionShift struct {
	st *State
}

func (a *ActionShift) String() string {
	return fmt.Sprintf("{shift:%v}", a.st.id)
}

type ActionReduce struct {
	prod Rule // reduce to rule
	popN int  // pop n
}

func (a *ActionReduce) String() string {
	return fmt.Sprintf("{reduce:%v,pop=%v}", a.prod.id(), a.popN)
}

type ActionAccept struct {
}

func (a *ActionAccept) String() string {
	return fmt.Sprintf("{accept}")
}
