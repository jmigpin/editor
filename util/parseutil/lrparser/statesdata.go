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
		return nil, err
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
						prod:         rd.prod,
						popN:         rd.popLen(),
						prodCanBeNil: ruleCanBeNil(rd.prod),
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
		case *StringRule: // ex: keywords
			return 1, string(t.runes)
		case *StringMidRule: // ex: keywords
			return 2, string(t.runes)
		case *StringOrRule: // individual runes
			return 3, string(t.runes)
		case *FuncRule:
			return 5, t.name
			//return -1, t.name
			// TODO: FlagRule? boolRule?
		case *SingletonRule:
			switch t {
			case endRule:
				return 10, ""
			case anyruneRule: // last to parse
				return 20, ""
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
	// strings conflicts
	for _, st := range sd.states {
		m := map[rune]Rule{}
		for _, r := range st.rsetSorted {
			if r2, ok := r.(*StringOrRule); ok {
				for _, ru := range r2.runes {
					r3, ok := m[ru]
					if !ok {
						m[ru] = r
						continue
					}
					if r3 == r {
						return fmt.Errorf("%v: duplicated rune %q in rule %v", st.id, ru, r3)
					} else {
						return fmt.Errorf("%v: rune %q in %v is already in rset %v", st.id, ru, r, r3)
					}
				}
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
	prod         Rule // reduce to rule
	popN         int  // pop n
	prodCanBeNil bool
	//prodIsLoop   bool
}

func (a *ActionReduce) String() string {
	return fmt.Sprintf("{reduce:%v,pop=%v}", a.prod.id(), a.popN)
}

type ActionAccept struct {
}

func (a *ActionAccept) String() string {
	return fmt.Sprintf("accept")
}
