package lrparser

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/jmigpin/editor/util/goutil"
)

type ContentParser struct {
	Opt          *CpOpt
	vd           *VerticesData
	sd           *StatesData
	buildNodeFns map[Rule]BuildNodeFn
}

func newContentParser(opt *CpOpt, ri *RuleIndex) (*ContentParser, error) {
	cp := &ContentParser{Opt: opt}
	cp.buildNodeFns = map[Rule]BuildNodeFn{}

	vd, err := newVerticesData(ri, cp.Opt.StartRule, cp.Opt.Reverse)
	if err != nil {
		return nil, err
	}
	cp.vd = vd

	sd, err := newStatesData(vd, cp.Opt.ShiftOnSRConflict)
	if err != nil {
		if sd != nil {
			err = fmt.Errorf("%w\n%v\n%v\n%v", err, ri, vd.rFirst, sd)
		}
		return nil, err
	}
	cp.sd = sd

	return cp, nil
}

//----------

func (cp *ContentParser) Parse(src []byte, index int) (*BuildNodeData, *cpRun, error) {
	fset := NewFileSetFromBytes(src)
	return cp.ParseFileSet(fset, index, nil)
}
func (cp *ContentParser) ParseFileSet(fset *FileSet, index int, extData any) (*BuildNodeData, *cpRun, error) {
	ps := NewPState(fset.Src)
	ps.Pos = index
	ps.Sc.Reverse = cp.Opt.Reverse
	cpr := newCPRun(cp.Opt, ps)
	cpr.externalData = extData
	cpn, err := cp.parse3(cpr)
	if err != nil {
		pe := &PosError{Err: err, Pos: cpr.ps.Pos}
		err = fset.Error(pe)
		if cpr.opt.VerboseError {
			err = fmt.Errorf("%w\n%s", err, cpr.Debug(cp))
		}
		return nil, cpr, err
	}
	d := newBuildNodeData(cpr, cpn)
	return d, cpr, nil
}

//----------

func (cp *ContentParser) parse3(cpr *cpRun) (*CPNode, error) {
	// add initial state to stack
	cpn0 := newCPNode(cpr.ps.Pos, cpr.ps.Pos, nil)
	item0 := &cpsItem{st: cp.sd.states[0], cpn: cpn0}
	cpr.stk = cpStack{item0}
	cpr.logf("%v\n", cpr.stk)
	// first input (action rule)
	prule, err := cp.nextParseRule(cpr, item0.st)
	if err != nil {
		return nil, err
	}
	// run forever
	for {
		item := cpr.stk[len(cpr.stk)-1] // stack top

		as := item.st.action[prule]
		// TODO: deal with this error at statesdata build time?
		if len(as) != 1 {
			return nil, fmt.Errorf("expected one action for %v, got %v (st=%v)", prule, as, item.st.id)
		}
		a := as[0]

		switch t := a.(type) {
		case *ActionShift:
			prule, err = cp.shift(cpr, t)
			if err != nil {
				return nil, err
			}
		case *ActionReduce:
			if err := cp.reduce(cpr, t); err != nil {
				return nil, err
			}
		case *ActionAccept:
			// handle earlystop (nodes with errors)
			if item.cpn.simulated {
				return nil, cpr.earlyStop.err
			}

			return item.cpn, nil
		default:
			return nil, goutil.TodoError()
		}
	}
}
func (cp *ContentParser) shift(cpr *cpRun, t *ActionShift) (Rule, error) {
	// correct simulated node position
	cpn := cpr.ps.Node.(*CPNode)
	if cpn.simulated {
		i := cpr.stk.topEnd()
		cpn.SetPos(i, i)
	}

	cpr.logf("shift %v\n", t.st.id)
	item := &cpsItem{st: t.st, cpn: cpn}
	cpr.stk = append(cpr.stk, item)
	cpr.logf("%v\n", cpr.stk)

	if err := cp.buildNode(cpr, cpn.rule, cpn); err != nil {
		return nil, err
	}

	// next input
	return cp.nextParseRule(cpr, t.st)
}
func (cp *ContentParser) reduce(cpr *cpRun, ar *ActionReduce) error {
	if cpr.isLogging() { // performance
		cpr.logf("reduce to %v (pop %v)\n", ar.prod.id(), ar.popN)
	}

	// pop n items
	popPos := len(cpr.stk) - ar.popN
	pops := cpr.stk[popPos:]
	cpr.stk = cpr.stk[:popPos] // pop

	// use current stk top to find the rule transition
	item3 := cpr.stk[len(cpr.stk)-1] // top of stack
	st2, ok := item3.st.gotoSt[ar.prod]
	if !ok {
		return fmt.Errorf("no goto for rule %v in %v ", ar.prod.id(), item3.st.id)
	}
	cpn, err := cp.groupPopped(cpr, ar, pops)
	if err != nil {
		return err
	}
	item4 := &cpsItem{st: st2, cpn: cpn}
	cpr.stk = append(cpr.stk, item4) // push "goto" to stk
	cpr.logf("%v\n", cpr.stk)

	return cp.buildNode(cpr, ar.prod, cpn)
}

//----------

func (cp *ContentParser) buildNode(cpr *cpRun, r Rule, cpn *CPNode) error {
	if cpn.simulated {
		return nil
	}
	fn, ok := cp.buildNodeFns[r]
	if !ok {
		return nil
	}
	d := newBuildNodeData(cpr, cpn)
	return fn(d)
}

//----------

func (cp *ContentParser) groupPopped(cpr *cpRun, ar *ActionReduce, pops []*cpsItem) (*CPNode, error) {
	cpn := cp.groupPopped2(cpr, ar, pops)
	cp.propagateSimulatedAndRecover(cpr, ar, cpn)
	return cpn, nil
}
func (cp *ContentParser) groupPopped2(cpr *cpRun, ar *ActionReduce, pops []*cpsItem) *CPNode {
	if len(pops) == 0 { // handle no pops reductions (nil rules)
		i := cpr.stk.topEnd()
		cpn := newCPNode(i, i, ar.prod)
		return cpn
	} else {
		// group popped items nodes into one node
		w := make([]*CPNode, 0, len(pops))
		for _, item2 := range pops {
			w = append(w, item2.cpn)
		}
		cpn := newCPNode2(w[0], w[len(w)-1], ar.prod)
		isReverse := cp.Opt.Reverse && ruleProdCanReverse(ar.prod)
		cpn.addChilds(isReverse, w...)
		return cpn
	}
}
func (cp *ContentParser) propagateSimulatedAndRecover(cpr *cpRun, ar *ActionReduce, cpn *CPNode) {
	simulatedChilds := false
	for _, cpn2 := range cpn.childs {
		if cpn2.simulated {
			simulatedChilds = true
			break
		}
	}
	if !simulatedChilds {
		return
	}

	// attempt to recover simulated childs
	if dr, ok := cpn.rule.(*DefRule); ok {
		if dr.isPOptional {
			cpn.childs = nil
			cpn.SetPos(cpn.Pos(), cpn.Pos()) // clear end (as if empty)
			cpr.logf("recovered: optional\n")
			return
		}
		if dr.isPZeroOrMore {
			*cpn = *cpn.childs[0]
			cpr.logf("recovered: pZeroOrMore\n")
			return
		}
		if dr.isPOneOrMore {
			if !cpn.childs[0].PosEmpty() {
				cpn.childs = cpn.childs[0].childs
				cpr.logf("recovered: pOneOrMore\n")
				return
			}
		}
	}

	// simulated
	cpn.simulated = true
	cpn.childs = nil
	cpn.SetPos(cpn.Pos(), cpn.Pos()) // clear end (as if empty)
}

//----------

func (cp *ContentParser) nextParseRule(cpr *cpRun, st *State) (Rule, error) {
	cpr.logf("rset: %v\n", st.rsetSorted)

	if cpr.earlyStop.on {
		return cp.simulateParseRuleSet(cpr, st)
	}

	r, err := cp.parseRuleSet(cpr, st.rsetSorted)
	if err == nil {
		return r, nil
	}

	// allow input to not be fully consumed
	if cp.Opt.EarlyStop {
		cpr.logf("earlystop: %v\n", err)
		cpr.earlyStop.on = true
		cpr.earlyStop.err = &PosError{Err: err, Pos: cpr.ps.Pos}
		return cp.simulateParseRuleSet(cpr, st)
	}

	return nil, err
}

//----------

func (cp *ContentParser) simulateParseRuleSet(cpr *cpRun, st *State) (Rule, error) {
	// rule to simulate
	r := (Rule)(nil)
	if st.rsetHasEndRule { // performance: faster stop (not necessary)
		r = endRule
	} else {
		if len(st.rsetSorted) == 0 {
			return nil, fmt.Errorf("empty rset to simulate")
		}

		// get index to try next
		k := cpr.earlyStop.simStateRsetIter[st] % len(st.rsetSorted)
		cpr.earlyStop.simStateRsetIter[st]++
		maxIter := 20
		if cpr.earlyStop.simStateRsetIter[st] >= maxIter {
			return nil, fmt.Errorf("reached max simulated attempts: %v; %w", maxIter, cpr.earlyStop.err)
		}

		r = st.rsetSorted[k]
	}

	i := cpr.stk.topEnd()
	cpn := newCPNode(i, i, r)
	cpn.simulated = true
	cpr.ps.Node = cpn
	if cpr.isLogging() { // performance
		cpr.logf("simulate parseruleset: %v %v\n", r.id(), PNodePosStr(cpn))
	}

	return r, nil
}

//----------

// creates a cpnode in ps
func (cp *ContentParser) parseRuleSet(cpr *cpRun, rset []Rule) (Rule, error) {
	for _, r := range rset {
		if err := cp.parseRule(cpr.ps, r); err != nil {
			continue
		}
		if cpr.isLogging() { // performance
			cpr.logf("parseruleset: %v %v\n", r.id(), PNodePosStr(cpr.ps.Node))
		}
		return r, nil
	}
	return nil, fmt.Errorf("failed to parse next: %v", rset)
}

func (cp *ContentParser) parseRule(ps *PState, r Rule) error {
	switch t := r.(type) {
	case *StringRule:
		pos0 := ps.Pos
		if err := t.parse(ps); err != nil {
			return err
		}
		ps.Node = newCPNode(pos0, ps.Pos, t)
	case *FuncRule:
		pos0 := ps.Pos
		if err := t.fn(ps); err != nil {
			ps.Pos = pos0
			return err
		}
		ps.Node = newCPNode(pos0, ps.Pos, t)
	case *SingletonRule:
		switch t {
		//case nilRule:	// commented: not called to be parsed
		case endRule:
			pos0 := ps.Pos
			if p2, err := ps.Sc.M.Eof(ps.Pos); err != nil {
				ps.Pos = p2
				return fmt.Errorf("not eof")
			}
			ps.Node = newCPNode(pos0, ps.Pos, t)
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
type CpOpt struct {
	StartRule         string // can be empty, will try to get it from grammar
	VerboseError      bool
	EarlyStop         bool // artificially parses an endrule when nextparsedrule fails. Allows parsing to stop successfully when no more input is recognized (although there is still input), while the rules are still able to reduce correctly.
	ShiftOnSRConflict bool
	Reverse           bool // runs input/rules in reverse (useful to backtrack in the middle of big inputs to then parse normally)
}

//----------
//----------
//----------

type cpRun struct {
	opt       *CpOpt
	ps        *PState
	stk       cpStack
	earlyStop struct {
		on               bool
		err              error
		simStateRsetIter map[*State]int // iterate over state rset rules to avoid repeating simulated
	}
	logBuf       bytes.Buffer
	externalData any
}

func newCPRun(opt *CpOpt, ps *PState) *cpRun {
	cpr := &cpRun{opt: opt, ps: ps}
	cpr.earlyStop.simStateRsetIter = map[*State]int{}
	return cpr
}
func (cpr *cpRun) isLogging() bool {
	return cpr.opt.VerboseError
}
func (cpr *cpRun) logf(f string, args ...any) {
	if cpr.isLogging() {
		fmt.Fprintf(&cpr.logBuf, f, args...)
	}
}
func (cpr *cpRun) Debug(cp *ContentParser) string {
	return fmt.Sprintf("%s\n%s\n%s\n%s%s",
		cp.vd.rFirst.ri,
		cp.vd.rFirst,
		cp.vd,
		cp.sd,
		bytes.TrimSpace(cpr.logBuf.Bytes()),
	)
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

//godebug:annotateoff
func (stk cpStack) String() string {
	u := []string{}
	for _, item := range stk {
		s := fmt.Sprintf("%v:", item.st.id)
		if item.cpn != nil { // can be nil in state0
			if item.cpn.rule != nil { // can be nil in state0
				s += fmt.Sprintf(" %v", item.cpn.rule.id())
			}
			s += " " + PNodePosStr(item.cpn)
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
	//simulated bool // TODO: move cpn.simulated here
}

//----------
//----------
//----------

func indentStr(t string, u string) string {
	u = strings.TrimRight(u, "\n")
	u = t + strings.ReplaceAll(u, "\n", "\n"+t) + "\n"
	return u
}
