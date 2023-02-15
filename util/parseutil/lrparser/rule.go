package lrparser

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/jmigpin/editor/util/goutil"
	"github.com/jmigpin/editor/util/parseutil/pscan"
)

type Rule interface {
	id() string
	isTerminal() bool
	childs() []Rule
	iterChildRefs(fn func(index int, ref *Rule) error) error
	String() string
}

//----------
//----------
//----------

// common rule
type CmnRule struct {
	childs2 []Rule
}

//----------

func (r *CmnRule) addChilds(r2 ...Rule) {
	r.childs2 = append(r.childs2, r2...)
}
func (r *CmnRule) onlyChild() Rule {
	return r.childs2[0]
}
func (r *CmnRule) setOnlyChild(r2 Rule) {
	r.childs2 = r.childs2[:0]
	r.addChilds(r2)
}

//----------

//godebug:annotateoff
func (r *CmnRule) iterChildRefs(fn func(index int, ref *Rule) error) error {
	for i := 0; i < len(r.childs2); i++ {
		if err := fn(i, &r.childs2[i]); err != nil {
			return err
		}
	}
	return nil
}
func (r *CmnRule) childs() []Rule {
	return r.childs2
}

//----------
//----------
//----------

// definition rule
// (1 child)
type DefRule struct {
	BasicPNode
	CmnRule
	name      string
	isStart   bool // has "start" symbol in the grammar
	isNoPrint bool // don't print in rule index (useful for tests)

	// specially handled cases
	isNoReverse   bool // don't reverse child sequence in reverse mode
	isPOptional   bool // parenthesis: optional
	isPZeroOrMore bool // parenthesis: zeroormore
	isPOneOrMore  bool // parenthesis: oneormore
}

func (r *DefRule) isTerminal() bool {
	return false
}
func (r *DefRule) id() string {
	s := ""
	if r.isStart {
		s += defRuleStartSym
	}
	return fmt.Sprintf("%v%v", s, r.name)
}
func (r *DefRule) String() string {
	return fmt.Sprintf("%v = %v", r.id(), r.onlyChild().id())
}

var defRuleStartSym = "^"   // used in grammar
var defRuleNoPrintSym = "§" // used in grammar

//----------

// reference to a rule // replaced in dereference phase
// (0 childs)
type RefRule struct {
	BasicPNode
	CmnRule
	name string
}

func (r *RefRule) isTerminal() bool {
	return false
}
func (r *RefRule) id() string {
	return fmt.Sprintf("{r:%v}", r.name)
}
func (r *RefRule) String() string {
	return r.id()
}

//----------

// (n childs as a sequence, not productions)
type AndRule struct {
	BasicPNode
	CmnRule
}

func (r *AndRule) isTerminal() bool {
	return false
}
func (r *AndRule) id() string {
	w := []string{}
	for _, r := range r.childs2 {
		w = append(w, r.id())
	}
	u := strings.Join(w, " ")
	return fmt.Sprintf("[%v]", u)
}
func (r *AndRule) String() string {
	return r.id()
}

//----------

// (n childs)
type OrRule struct {
	BasicPNode
	CmnRule
}

func (r *OrRule) isTerminal() bool {
	return false
}
func (r *OrRule) id() string {
	w := []string{}
	for _, r := range r.childs2 {
		w = append(w, r.id())
	}
	u := strings.Join(w, "|")
	return fmt.Sprintf("[%v]", u)
}
func (r *OrRule) String() string {
	return r.id()
}

//----------

// replaced in dereference phase
// (3 childs: [conditional,then,else])
type IfRule struct {
	BasicPNode
	CmnRule
}

func (r *IfRule) selfSequence() []Rule { return []Rule{r} }
func (r *IfRule) isTerminal() bool     { return false }
func (r *IfRule) id() string {
	return fmt.Sprintf("{if %v ? %v : %v}", r.childs2[0], r.childs2[1], r.childs2[2])
}
func (r *IfRule) String() string {
	return r.id()
}

//----------

// To be used in src code and then found by IfRule; the value is observed when building the contentparser, not at parse time
// (0 childs)
type BoolRule struct {
	CmnRule
	name  string
	value bool
}

func (r *BoolRule) isTerminal() bool {
	return true
}
func (r *BoolRule) id() string {
	return fmt.Sprintf("{b:%v:%v}", r.name, r.value)
}
func (r *BoolRule) String() string {
	return r.id()
}

//----------

// parenthesis, ex: (aaa (bbb|ccc))
// replaced by defrules at ruleindex
// (1 childs)
type ParenRule struct {
	BasicPNode
	CmnRule
	typ parenRType
}

func (r *ParenRule) isTerminal() bool {
	return false
}

func (r *ParenRule) id() string {
	s := ""
	if r.typ != parenRTNone {
		s = string(r.typ)
	}
	return fmt.Sprintf("(%v)%v", r.onlyChild().id(), s)
}
func (r *ParenRule) String() string {
	return r.id()
}

//----------

// (0 childs)
type StringRule struct {
	BasicPNode
	CmnRule
	runes   []rune
	rranges []pscan.RuneRange
	typ     stringRType
}

func (r *StringRule) isTerminal() bool {
	return true
}
func (r *StringRule) id() string {
	s := ""
	if len(r.runes) > 0 {
		s += fmt.Sprintf("%q", string(r.runes))
	}
	if len(r.rranges) > 0 {
		u := []string{}
		if len(s) > 0 {
			u = append(u, s)
		}
		for _, rr := range r.rranges {
			u = append(u, fmt.Sprintf("%v", rr))
		}
		s = strings.Join(u, ",")
		return fmt.Sprintf("{%v,%v}", s, r.typ)
	}
	return fmt.Sprintf("%v%v", s, r.typ)
}
func (r *StringRule) String() string {
	return r.id()
}

//----------

func (sr1 *StringRule) intersect(sr2 *StringRule) (bool, error) {
	switch sr1.typ {
	case stringRTOr, stringRTOrNeg:
	default:
		return false, fmt.Errorf("expecting or/orneg: %T", sr1.typ)
	}
	switch sr2.typ {
	case stringRTOr, stringRTOrNeg:
	default:
		return false, fmt.Errorf("expecting or/orneg: %T", sr2.typ)
	}
	if sr1.typ == sr2.typ { // same polarity
		if sr1.intersectRunes(sr2.runes) {
			return true, nil
		}
		if sr1.intersectRanges(sr2.rranges) {
			return true, nil
		}
	} else {
		if !sr1.intersectRunes(sr2.runes) && !sr1.intersectRanges(sr2.rranges) {
			return true, nil
		}
	}
	return false, nil
}
func (r *StringRule) intersectRunes(rus []rune) bool {
	for _, ru2 := range rus {
		for _, ru := range r.runes {
			if ru == ru2 {
				return true
			}
		}
		for _, rr := range r.rranges {
			if rr.HasRune(ru2) {
				return true
			}
		}
	}
	return false
}
func (r *StringRule) intersectRanges(rrs []RuneRange) bool {
	for _, rr2 := range rrs {
		for _, ru := range r.runes {
			if rr2.HasRune(ru) {
				return true
			}
		}
		for _, rr := range r.rranges {
			if rr.IntersectsRange(rr2) {
				return true
			}
		}
	}
	return false
}

//----------

func (r *StringRule) parse(ps *PState) error {
	if p2, err := r.parse2(ps.Pos, ps.Sc); err != nil {
		return err
	} else {
		ps.Pos = p2
		return nil
	}
}
func (r *StringRule) parse2(pos int, sc *pscan.Scanner) (int, error) {
	switch r.typ {
	case stringRTAnd: // sequence, ex: keyword
		return sc.M.RuneSequence(pos, r.runes)
	case stringRTMid: // sequence, ex: keyword
		return sc.M.RuneSequenceMid(pos, r.runes)
	case stringRTOr:
		return sc.M.Or(pos,
			sc.W.RuneOneOf(r.runes),
			sc.W.RuneRanges(r.rranges...),
		)
	case stringRTOrNeg:
		return sc.M.And(pos,
			sc.W.MustErr(sc.W.Or(
				sc.W.RuneOneOf(r.runes),
				sc.W.RuneRanges(r.rranges...),
			)),
			sc.M.OneRune,
		)
	default:
		panic(goutil.TodoErrorStr(string(r.typ)))
	}
}

//----------

// processor function call rule: allows processing rules at compile time. Ex: string operations.
// (0 childs)
type ProcRule struct {
	BasicPNode
	CmnRule
	name string
	args []ProcRuleArg // allows more then just rules (ex: ints)
}

func (r *ProcRule) isTerminal() bool {
	return true
}
func (r *ProcRule) id() string {
	return fmt.Sprintf("%v(%v)", r.name, r.childs())
}
func (r *ProcRule) String() string {
	return r.id()
}

//----------

// (0 childs)
type FuncRule struct {
	CmnRule
	name       string
	parseOrder int // value for sorting parse order, zero for func default, check
	fn         PStateParseFn
}

func (r *FuncRule) isTerminal() bool {
	return true
}
func (r *FuncRule) id() string {
	sv := ""
	if r.parseOrder != 0 {
		sv = fmt.Sprintf("<%v>", r.parseOrder)
	}
	return fmt.Sprintf("%v%v", r.name, sv)
}
func (r *FuncRule) String() string {
	return r.id()
}

//----------

// (0 childs)
type SingletonRule struct {
	BasicPNode
	CmnRule
	name   string
	isTerm bool
}

func newSingletonRule(name string, isTerm bool) *SingletonRule {
	return &SingletonRule{name: name, isTerm: isTerm}
}
func (r *SingletonRule) isTerminal() bool {
	return r.isTerm
}
func (r *SingletonRule) id() string     { return r.name }
func (r *SingletonRule) String() string { return r.id() }

// setup to be available in the grammars at ruleindex.go
var endRule = newSingletonRule("$", true)
var nilRule = newSingletonRule("nil", true)

// special start rule to know start/end (not a terminal)
var startRule = newSingletonRule("^^^", false)

//----------
//----------
//----------

// parenthesis rule type
type parenRType rune

const (
	parenRTNone       parenRType = 0
	parenRTOptional   parenRType = '?'
	parenRTZeroOrMore parenRType = '*'
	parenRTOneOrMore  parenRType = '+'

	// strings related
	parenRTStrOr      parenRType = '%' // individual runes
	parenRTStrOrNeg   parenRType = '!' // individual runes: not
	parenRTStrOrRange parenRType = '-' // individual runes: range
	parenRTStrMid     parenRType = '~' // sequence: middle match
)

//----------

// string rule type
type stringRType byte

const (
	stringRTAnd stringRType = iota
	stringRTOr
	stringRTOrNeg
	stringRTMid
)

func (srt stringRType) String() string {
	switch srt {
	case stringRTAnd:
		return "" // empty
	case stringRTOr:
		return string(parenRTStrOr)
	case stringRTOrNeg:
		return string(parenRTStrOrNeg)
	case stringRTMid:
		return string(parenRTStrMid)
	default:
		panic(srt)
	}
}

// ----------

type ProcRuleFn func(args ProcRuleArgs) (Rule, error)
type ProcRuleArg any
type ProcRuleArgs []ProcRuleArg

func (args ProcRuleArgs) Int(i int) (int, error) {
	if i >= len(args) {
		return 0, fmt.Errorf("missing arg %v", i)
	}
	arg := args[i]
	u, ok := arg.(int)
	if !ok {
		return 0, fmt.Errorf("arg %v is not an int (%T)", i, arg)
	}
	return u, nil
}
func (args ProcRuleArgs) Rule(i int) (Rule, error) {
	if i >= len(args) {
		return nil, fmt.Errorf("missing arg %v", i)
	}
	arg := args[i]
	u, ok := arg.(Rule)
	if !ok {
		return nil, fmt.Errorf("arg %v is not a rule (%T)", i, arg)
	}
	return u, nil
}
func (args ProcRuleArgs) MergedStringRule(i int) (*StringRule, error) {
	r, err := args.Rule(i)
	if err != nil {
		return nil, err
	}
	sr, err := mergeStringRules(r)
	if err != nil {
		return nil, fmt.Errorf("arg %v: %w", i, err)
	}
	return sr, nil
}

//----------
//----------
//----------

type RuleSet map[Rule]struct{}

func (rs RuleSet) set(r Rule) {
	rs[r] = struct{}{}
}
func (rs RuleSet) unset(r Rule) {
	delete(rs, r)
}
func (rs RuleSet) has(r Rule) bool {
	_, ok := rs[r]
	return ok
}
func (rs RuleSet) add(rs2 RuleSet) {
	for r := range rs2 {
		rs.set(r)
	}
}
func (rs RuleSet) remove(rs2 RuleSet) {
	for r := range rs2 {
		rs.unset(r)
	}
}
func (rs RuleSet) toSlice() []Rule {
	w := []Rule{}
	for r := range rs {
		w = append(w, r)
	}
	return w
}
func (rs RuleSet) sorted() []Rule {
	w := rs.toSlice()
	sortRules(w)
	return w
}
func (rs RuleSet) String() string {
	u := []string{}
	w := rs.sorted()
	for _, r := range w {
		u = append(u, fmt.Sprintf("%v", r))
	}
	return fmt.Sprintf("[%v]", strings.Join(u, ","))
}

//----------

func sortRuleSetForParse(rset RuleSet) []Rule {
	// integer/string for sorting
	svalues := func(r Rule) (int, string) {
		switch t := r.(type) {
		case *FuncRule:
			sv := 100 // allows grammars to use (1,2,...) value without thinking about the funcs default value
			if t.parseOrder != 0 {
				sv = t.parseOrder
			}
			return sv, t.name
		case *StringRule:
			switch t.typ {
			case stringRTAnd: // ex: keywords
				return 201, string(t.runes)
			case stringRTMid: // ex: keywords
				return 202, string(t.runes)
			case stringRTOr: // individual runes
				return 203, string(t.runes)
			case stringRTOrNeg: // individual runes
				return 204, string(t.runes)
			default:
				panic(goutil.TodoErrorStr(string(t.typ)))
			}
		case *SingletonRule:
			switch t {
			case endRule:
				return 301, ""
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
//----------
//----------

func sortRules(w []Rule) {
	sort.Slice(w, func(a, b int) bool {
		ra, rb := w[a], w[b]
		ta, sa := sortRulesValue(ra)
		tb, sb := sortRulesValue(rb)
		if ta == tb {
			return sa < sb
		}
		return ta < tb
	})
}
func sortRulesValue(r Rule) (int, string) {
	id := r.id()
	// terminals (last)
	if r.isTerminal() {
		return 5, id
	}
	// productions: start rule (special)
	if r == startRule {
		return 1, id
	}
	// productions: starting rule (grammar)
	if dr, ok := r.(*DefRule); ok && dr.isStart {
		return 2, id
	}
	// productions: name starts with a letter (as opposed to ex: "(")
	u := []rune(id)
	if unicode.IsLetter(u[0]) {
		return 3, id
	}
	// productions
	return 4, id
}

//----------
//----------
//----------

//godebug:annotateoff
func ruleProductions(r Rule) []Rule {
	switch t := r.(type) {
	case *AndRule: // andrule childs are not productions
		return []Rule{t}
	case *DefRule:
		switch t2 := t.onlyChild().(type) {
		case *OrRule:
			return t2.childs()
		}
	}
	return r.childs()
}

//godebug:annotateoff
func ruleSequence(r Rule, reverse bool) []Rule {
	switch t := r.(type) {
	case *AndRule: // andrule is the only rule whose childs provide a sequence
		if reverse {
			// use a copy to avoid changing the original rule that could be used for other grammars that are non-reverse
			return reverseRulesCopy(t.childs())
		}
		return t.childs()
	default:
		return []Rule{t}
	}
}
func ruleProdCanReverse(r Rule) bool {
	if dr, ok := r.(*DefRule); ok {
		return !dr.isNoReverse
	}
	return true
}

//func ruleIsLoop(r Rule) bool {
//	dr, ok := r.(*DefRule)
//	return ok && dr.isLoop
//}
//func ruleCanBeNil(r0 Rule) bool {
//	seen := map[Rule]bool{}
//	vis := (func(r Rule) bool)(nil)
//	vis = func(r Rule) bool {
//		if seen[r] {
//			return false
//		}
//		seen[r] = true
//		defer func() { seen[r] = false }()

//		if r == nilRule {
//			return true
//		}
//		switch t := r.(type) {
//		case *DefRule:
//			return vis(t.onlyChild())
//		case *OrRule:
//			for _, r2 := range t.childs2 {
//				if vis(r2) {
//					return true
//				}
//			}
//		case *AndRule:
//			for _, r2 := range t.childs2 {
//				if !seen[r2] && !vis(r2) {
//					return false
//				}
//			}
//			return true
//		}
//		return false
//	}
//	return vis(r0)
//}

//----------

func mergeStringRules(r Rule) (*StringRule, error) {
	switch t := r.(type) {
	case *StringRule:
		return t, nil
	case *DefRule:
		sr, err := mergeStringRules(t.onlyChild())
		if err != nil {
			// improve error
			err = fmt.Errorf("%v: %w", t.name, err)
		}
		return sr, err
	case *OrRule:
		// concat "or" rules
		sr2 := &StringRule{typ: stringRTOr}
		for _, c := range t.childs() {
			if sr3, err := mergeStringRules(c); err != nil {
				return nil, err
			} else {
				switch sr3.typ {
				case stringRTOr:
					sr2.runes = append(sr2.runes, sr3.runes...)
					sr2.rranges = append(sr2.rranges, sr3.rranges...)
				default:
					return nil, fmt.Errorf("unable to merge %v from %v into orrule", sr3, t)
				}
			}
		}
		return sr2, nil
	case *AndRule:
		// concat "and" rules
		sr2 := &StringRule{typ: stringRTAnd}
		for _, c := range t.childs() {
			if sr3, err := mergeStringRules(c); err != nil {
				return nil, err
			} else {
				switch sr3.typ {
				case stringRTAnd:
					sr2.runes = append(sr2.runes, sr3.runes...)
					sr2.rranges = append(sr2.rranges, sr3.rranges...)
				default:
					return nil, fmt.Errorf("unable to merge %v from %v into andrule", sr3, r)
				}
			}
		}
		return sr2, nil
	default:
		return nil, fmt.Errorf("unable to merge to stringrule: %T, %v", r, r)
	}
}

//----------

func reverseRulesCopy(w []Rule) []Rule {
	u := make([]Rule, len(w))
	copy(u, w)
	reverseRules(u)
	return u
}
func reverseRules(w []Rule) {
	l := len(w)
	for i := 0; i < l/2; i++ {
		k := l - 1 - i
		w[i], w[k] = w[k], w[i]
	}
}

//----------

func walkRuleChilds(rule Rule, fn func(*Rule) error) error {
	return rule.iterChildRefs(func(index int, ref *Rule) error {
		return fn(ref)
	})
}

//----------
//----------
//----------

// TODO: rename
type PStateParseFn func(ps *PState) error

//----------
//----------
//----------

//type RuleProductions []RuleSequence

//func (rp RuleProductions) String() string {
//	w := []string{}
//	for _, rs := range rp {
//		w = append(w, rs.String())
//	}
//	u := strings.Join(w, " | ")
//	return fmt.Sprintf("[%v]", u)
//}

////----------

//type RuleSequence []Rule

//func (rs RuleSequence) String() string {
//	w := []string{}
//	for _, r := range rs {
//		w = append(w, r.String())
//	}
//	u := strings.Join(w, " ")
//	return fmt.Sprintf("[%v]", u)
//}
