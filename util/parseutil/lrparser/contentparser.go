package lrparser

import (
	"fmt"
	"strings"

	"github.com/jmigpin/editor/util/goutil"
)

type ContentParser struct {
	Opt          *CPOpt
	vd           *VerticesData
	sd           *StatesData
	buildNodeFns map[Rule]BuildNodeFn

	stk cpStack
	// run vars (need reset for each parse)
	earlyStop struct {
		on               bool
		err              error
		simStateRsetIter map[*State]int // iterate over state rset rules to avoid repeating simulated
	}
}

func newContentParser(opt *CPOpt, ri *RuleIndex) (*ContentParser, error) {
	cp := &ContentParser{Opt: opt}
	cp.buildNodeFns = map[Rule]BuildNodeFn{}

	vd, err := newVerticesData(ri, cp.Opt.StartRule, cp.Opt.Reverse)
	if err != nil {
		return nil, err
	}
	cp.vd = vd
	cp.Opt.Logf("\n%v\n", ri) // dereferenced
	cp.Opt.Logf("\n%v\n", vd.rFirst)

	sd, err := newStatesData(vd, cp.Opt.ShiftOnSRConflict)
	if err != nil {
		//cp.Opt.Logf("%v\n", vd)
		return nil, err
	}
	cp.sd = sd

	return cp, nil
}

//----------

func (cp *ContentParser) Parse(src []byte, index int) (*BuildNodeData, error) {
	fset := NewFileSetFromBytes(src)
	return cp.ParseFileSet(fset, index)
}
func (cp *ContentParser) ParseFileSet(fset *FileSet, index int) (*BuildNodeData, error) {
	// DEBUG
	if cp.Opt.LogfFn != nil {
		cp.Opt.Logf(cp.vd.String())
		cp.Opt.Logf(cp.sd.String())
	}

	ps := &PState{src: fset.Src, i: index, reverse: cp.vd.reverse}
	cpn, err := cp.parse3(ps)
	if err != nil {
		pe := &PosError{err: err, Pos: ps.i}
		return nil, fset.Error(pe)
	}
	d := &BuildNodeData{ps: ps, cpn: cpn}
	return d, nil
}

//----------

func (cp *ContentParser) parse3(ps *PState) (*CPNode, error) {
	// init parse vars
	cp.earlyStop.on = false
	cp.earlyStop.err = nil
	cp.earlyStop.simStateRsetIter = map[*State]int{}
	// add initial state to stack
	cpn0 := newCPNode(ps.i, ps.i, nil)
	item0 := &cpsItem{st: cp.sd.states[0], cpn: cpn0}
	cp.stk = cpStack{item0}
	cp.Opt.Logf("%v\n", cp.stk)
	// first input (action rule)
	prule, err := cp.nextParseRule(ps, item0.st)
	if err != nil {
		return nil, err
	}
	// run forever
	for {
		item := cp.stk[len(cp.stk)-1] // stack top

		as := item.st.action[prule]
		// TODO: deal with this error at statesdata build time?
		if len(as) != 1 {
			return nil, fmt.Errorf("expected one action for %v, got %v", prule, as)
		}
		a := as[0]

		switch t := a.(type) {
		case *ActionShift:
			prule, err = cp.shift(ps, t)
			if err != nil {
				return nil, err
			}
		case *ActionReduce:
			if err := cp.reduce(ps, t); err != nil {
				return nil, err
			}
		case *ActionAccept:
			// handle earlystop (nodes with errors)
			if item.cpn.simulated {
				return nil, cp.earlyStop.err
			}

			return item.cpn, nil
		default:
			return nil, goutil.TodoError()
		}
	}
}
func (cp *ContentParser) shift(ps *PState, t *ActionShift) (Rule, error) {
	// correct simulated node position
	cpn := ps.parseNode.(*CPNode)
	if cpn.simulated {
		i := cp.stk.topEnd()
		cpn.setPos(i, i)
	}

	cp.Opt.Logf("shift %v\n", t.st.id)
	item := &cpsItem{st: t.st, cpn: cpn}
	cp.stk = append(cp.stk, item)
	cp.Opt.Logf("%v\n", cp.stk)

	// next input
	return cp.nextParseRule(ps, t.st)
}
func (cp *ContentParser) reduce(ps *PState, ar *ActionReduce) error {
	cp.Opt.Logf("reduce to %v (pop %v)\n", ar.prod.id(), ar.popN)

	// pop n items
	popPos := len(cp.stk) - ar.popN
	pops := cp.stk[popPos:]
	cp.stk = cp.stk[:popPos] // pop

	// use current stk top to find the rule transition
	item3 := cp.stk[len(cp.stk)-1] // top of stack
	st2, ok := item3.st.gotoSt[ar.prod]
	if !ok {
		return fmt.Errorf("no goto for rule %v in %v ", ar.prod.id(), item3.st.id)
	}
	cpn, err := cp.processPopped(ar, pops)
	if err != nil {
		return err
	}
	item4 := &cpsItem{st: st2, cpn: cpn}
	cp.stk = append(cp.stk, item4) // push "goto" to stk
	cp.Opt.Logf("%v\n", cp.stk)

	// build/alter node func
	if !cpn.simulated {
		if fn, ok := cp.buildNodeFns[ar.prod]; ok {
			d := &BuildNodeData{ps: ps, cpn: cpn}
			if err := fn(d); err != nil {
				return err
			}
		}
	}

	return nil
}

//----------

func (cp *ContentParser) processPopped(ar *ActionReduce, pops []*cpsItem) (*CPNode, error) {
	cpn := cp.processPopped2(ar, pops)
	cp.propagateSimulated(ar, cpn)
	return cpn, nil
}
func (cp *ContentParser) processPopped2(ar *ActionReduce, pops []*cpsItem) *CPNode {
	// reducing to a rule that is a loop (flatten childs list)
	if ruleIsLoop(ar.prod) && len(pops) == 2 {
		cpn0 := pops[0].cpn
		if cpn0.rule == ar.prod { // first popped item is the loop rule (this depends on how the loop was constructed, check ruleindex)
			cpn1 := pops[1].cpn

			// recover from error
			if cpn1.simulated {
				cp.Opt.Logf("recovered: loop node")
				return cpn0 // ignore the second popped item since it has an error
			}

			cpn2 := newCPNode2(cpn0, cpn1, ar.prod)
			cpn2.childs = cpn0.childs
			cpn2.addChilds(cp.vd.reverse, cpn1)
			return cpn2
		}
	}

	if len(pops) == 0 { // handle no pops reductions (nil rules)
		i := cp.stk.topEnd()
		cpn := newCPNode(i, i, ar.prod)
		return cpn
	} else {
		// group popped items nodes into one node
		w := []*CPNode{}
		for _, item2 := range pops {
			w = append(w, item2.cpn)
		}
		cpn := newCPNode2(w[0], w[len(w)-1], ar.prod)
		cpn.addChilds(cp.vd.reverse, w...)
		return cpn
	}
}
func (cp *ContentParser) propagateSimulated(ar *ActionReduce, cpn *CPNode) {
	simulated := false
	for _, cpn2 := range cpn.childs {
		if cpn2.simulated {
			simulated = true
		}
	}
	if simulated {
		cpn.simulated = true
		cpn.end = cpn.pos // clear end position (as if empty)

		// recover from error
		if ar.prodCanBeNil {
			cp.Opt.Logf("recovered: prod can be nil")
			cpn.childs = nil
			cpn.simulated = false
		}
	}
}

//----------

func (cp *ContentParser) nextParseRule(ps *PState, st *State) (Rule, error) {
	cp.Opt.Logf("rset: %v\n", st.rsetSorted)

	if cp.earlyStop.on {
		return cp.simulateParseRuleSet(ps, st)
	}

	r, err := cp.parseRuleSet(ps, st.rsetSorted)
	if err == nil {
		return r, nil
	}

	// allow input to not be fully consumed
	if cp.Opt.EarlyStop {
		cp.Opt.Logf("earlystop: %v\n", err)
		cp.earlyStop.on = true
		cp.earlyStop.err = &PosError{err: err, Pos: ps.i}
		return cp.simulateParseRuleSet(ps, st)
	}

	return nil, err
}

//----------

func (cp *ContentParser) simulateParseRuleSet(ps *PState, st *State) (Rule, error) {
	// rule to simulate
	r := (Rule)(nil)
	if st.rsetHasEndRule { // performance: faster stop (not necessary)
		r = endRule
	} else {
		// get index to try next
		k := cp.earlyStop.simStateRsetIter[st] % len(st.rsetSorted)
		cp.earlyStop.simStateRsetIter[st]++
		maxIter := 20
		if cp.earlyStop.simStateRsetIter[st] >= maxIter {
			return nil, fmt.Errorf("reached max simulated attempts: %v; %w", maxIter, cp.earlyStop.err)
		}

		r = st.rsetSorted[k]
	}

	i := cp.stk.topEnd()
	cpn := newCPNode(i, i, r)
	cpn.simulated = true
	ps.parseNode = cpn
	cp.Opt.Logf("simulate parseruleset: %v %v\n", r.id(), pnodePosStr(cpn))

	return r, nil
}

//----------

// creates a cpnode in ps
func (cp *ContentParser) parseRuleSet(ps *PState, rset []Rule) (Rule, error) {
	for _, r := range rset {
		if err := cp.parseRule(ps, r); err != nil {
			continue
		}
		cp.Opt.Logf("parseruleset: %v %v\n", r.id(), pnodePosStr(ps.parseNode))
		return r, nil
	}
	return nil, fmt.Errorf("failed to parse next: %v", rset)
}

func (cp *ContentParser) parseRule(ps *PState, r Rule) error {
	switch t := r.(type) {
	case *StringRule:
		i0 := ps.i
		switch t.typ {
		case stringrNone:
			if err := ps.MatchRunesAnd(t.runes); err != nil {
				return err
			}
		case stringrRunes:
			if err := ps.MatchRunesOr(t.runes); err != nil {
				return err
			}
		case stringrMidMatch:
			if err := ps.matchRunesMid(t.runes); err != nil {
				return err
			}
		default:
			panic(goutil.TodoErrorStr(string(t.typ)))
		}
		ps.parseNode = newCPNode(i0, ps.i, t)
	case *FuncRule:
		i0 := ps.i
		ps2 := ps.copy()
		if err := t.fn(ps2); err != nil {
			return err
		}
		ps.set(ps2)
		ps.parseNode = newCPNode(i0, ps.i, t)
	case *SingletonRule:
		switch t {
		// commented: should not be called to be parsed
		//case nilRule:
		//	ps.parseNode = newCPNode(ps.i, ps.i, t)

		case endRule: // not allowed in grammar ("$") but present in the rules to parse (rset/lookaheads)
			if err := ps.matchEof(); err != nil {
				return err
			}
			ps.parseNode = newCPNode(ps.i, ps.i, t)
		case anyruneRule:
			i0 := ps.i
			if _, err := ps.readRune(); err != nil {
				return err // fails at eof
			}
			ps.parseNode = newCPNode(i0, ps.i, t)
		default:
			panic(goutil.TodoErrorStr(t.name))
		}
	default:
		panic(goutil.TodoErrorType(t))
	}
	return nil
}

//----------

func (cp *ContentParser) SetBuildNodeFn(name string, buildFn BuildNodeFn) error {
	r, ok := cp.vd.rFirst.ri.get(name)
	if !ok {
		return fmt.Errorf("rule name not found: %v", name)
	}
	cp.buildNodeFns[r] = buildFn
	return nil
}

//----------
//----------
//----------

// content parser options
type CPOpt struct {
	EarlyStop         bool // artificially parses an endrule when nextparsedrule fails. Allows parsing to stop successfully when no more input is recognized (although there is still input), while the rules are still able to reduce correctly.
	ShiftOnSRConflict bool

	StartRule string // can be empty, will get it from grammar
	Reverse   bool   // runs input/rules in reverse (useful to backtrack in the middle of big inputs to then parse normally)

	HelperFn func()
	LogfFn   func(f string, a ...interface{})
}

func (opt *CPOpt) Logf(f string, a ...interface{}) {
	if opt.HelperFn != nil {
		opt.HelperFn()
	}
	if opt.LogfFn != nil {
		opt.LogfFn(f, a...)
	}
}

//----------
//----------
//----------

// content parser stack
type cpStack []*cpsItem

func (stk cpStack) topEnd() int {
	k := len(stk) - 1
	return stk[k].cpn.End()
}

func (stk cpStack) String() string {
	u := []string{}
	for _, item := range stk {
		s := fmt.Sprintf("%v:", item.st.id)
		if item.cpn != nil { // can be nil in state0
			if item.cpn.rule != nil { // can be nil in state0
				s += fmt.Sprintf(" %v", item.cpn.rule.id())
			}
			s += " " + pnodePosStr(item.cpn)
			if item.cpn.simulated {
				s += fmt.Sprintf(" (simulated)")
			}
		}
		u = append(u, s)
	}
	return fmt.Sprintf("stk{\n\t%v\n}", strings.Join(u, "\n\t"))
}

//----------

// content parser stack item
type cpsItem struct {
	st  *State
	cpn *CPNode
}

//----------
//----------
//----------

type simEntry struct {
	st *State
	r  Rule
}

//----------
//----------
//----------

func indentStr(t string, u string) string {
	u = strings.TrimRight(u, "\n")
	u = t + strings.ReplaceAll(u, "\n", "\n"+t) + "\n"
	return u
}
