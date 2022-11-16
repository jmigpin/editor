package lrparser

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jmigpin/editor/util/goutil"
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
	if err := sd.solveConflicts(vd); err != nil {
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
		for rd, _ := range v.rdslasK {
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

		st.rsetSorted = sd.sortForParse(rset)
		st.rsetHasEndRule = rset.has(endRule)
	}

	return nil
}

//----------

func (sd *StatesData) sortForParse(rset RuleSet) []Rule {
	// integer/string for sorting
	svalues := func(r Rule) (int, string) {
		switch t := r.(type) {
		case *StringRule:
			switch t.typ {
			case stringrAnd: // ex: keywords
				return 1, string(t.runes)
			case stringrMid: // ex: keywords
				return 2, string(t.runes)
			case stringrOr: // individual runes
				return 3, string(t.runes)
			case stringrNot: // individual runes
				return 4, string(t.runes)
			default:
				panic(goutil.TodoErrorStr(string(t.typ)))
			}
		case *FuncRule:
			return 10, t.name
		case *SingletonRule:
			switch t {
			case endRule:
				return 20, ""
			case anyruneRule: // last to parse
				return 21, ""
			}
			panic(goutil.TodoErrorStr(t.name))
		}
		panic(goutil.TodoErrorType(r))
	}

	x := rset.toSlice()
	sort.Slice(x, func(a, b int) bool {
		ra, rb := x[a], x[b]
		ta, sa := svalues(ra)
		tb, sb := svalues(rb)
		if ta == tb {
			return sa < sb
		}
		return ta < tb
	})
	return x
}

//----------

func (sd *StatesData) solveConflicts(vd *VerticesData) error {
	// strings conflicts: util func
	checkDuplicate := func(m map[rune]Rule, st *State, r Rule, ru rune) error {
		r2, ok := m[ru]
		if !ok {
			m[ru] = r
			return nil
		}
		return fmt.Errorf("%v: rune %q in %v is already defined at %v", st.id, ru, r, r2)
	}

	// strings conflicts (runes)
	for _, st := range sd.states {
		orM := map[rune]Rule{}
		notM := map[rune]Rule{}
		hasAnyrune := false

		// check duplicates
		for _, r := range st.rsetSorted {
			if r == anyruneRule {
				hasAnyrune = true
				continue
			}
			sr, ok := r.(*StringRule)
			if !ok {
				continue
			}
			switch sr.typ {
			case stringrAnd:
				if len(sr.runes) == 1 {
					if err := checkDuplicate(orM, st, r, sr.runes[0]); err != nil {
						return err
					}
				}
			case stringrOr:
				for _, ru := range sr.runes {
					if err := checkDuplicate(orM, st, r, ru); err != nil {
						return err
					}
				}
			case stringrNot:
				for _, ru := range sr.runes {
					if err := checkDuplicate(notM, st, r, ru); err != nil {
						return err
					}
				}
			}
		}

		// check conflicts: all "or" runes must be in "not"
		if len(notM) > 0 {
			for ru, r := range orM {
				_, ok := notM[ru]
				if !ok {
					// show "not" rules
					rs := &RuleSet{}
					for _, r2 := range notM {
						rs.set(r2)
					}

					return fmt.Errorf("%v: rune %q in %v is covered in %v", st.id, ru, r, rs)
				}
			}
		}
		if hasAnyrune {
			if len(orM) > 0 || len(notM) > 0 {
				return fmt.Errorf("%v: anyrune and stringrule in the same state\n%v", st.id, sd)
			}
		}
	}

	// action conflicts
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
