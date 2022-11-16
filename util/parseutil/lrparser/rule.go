package lrparser

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

type Rule interface {
	id() string

	isTerminal() bool

	childs() []Rule
	iterChildRefs(fn func(index int, ref *Rule) error) error

	String() string

	// TODO: consider
	// parse(*PState) error // for terminal rules
}

//----------
//----------
//----------

// common rule
type CmnRule struct {
	childs_ []Rule
}

//----------

func (r *CmnRule) addChild(r2 Rule) {
	r.childs_ = append(r.childs_, r2)
}
func (r *CmnRule) onlyChild() Rule {
	return r.childs_[0]
}
func (r *CmnRule) setOnlyChild(r2 Rule) {
	r.childs_ = r.childs_[:0]
	r.addChild(r2)
}

//----------

func (r *CmnRule) iterChildRefs(fn func(index int, ref *Rule) error) error {
	for i := 0; i < len(r.childs_); i++ {
		if err := fn(i, &r.childs_[i]); err != nil {
			return err
		}
	}
	return nil
}
func (r *CmnRule) childs() []Rule {
	return r.childs_
}

//----------
//----------
//----------

// definition rule
// (1 child)
type DefRule struct {
	CmnPNode
	CmnRule
	name    string
	isStart bool // has "start" symbol in the grammar

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

var defRuleStartSym = "^" // used in grammar

//----------

// reference to a rule
// replaced in dereference phase
// (0 childs)
type RefRule struct {
	CmnPNode
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
	CmnPNode
	CmnRule
}

func (r *AndRule) isTerminal() bool {
	return false
}
func (r *AndRule) id() string {
	w := []string{}
	for _, r := range r.childs_ {
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
	CmnPNode
	CmnRule
}

func (r *OrRule) isTerminal() bool {
	return false
}
func (r *OrRule) id() string {
	w := []string{}
	for _, r := range r.childs_ {
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
	CmnPNode
	CmnRule
}

func (r *IfRule) selfSequence() []Rule { return []Rule{r} }
func (r *IfRule) isTerminal() bool     { return false }
func (r *IfRule) id() string {
	return fmt.Sprintf("{if %v ? %v : %v}", r.childs_[0], r.childs_[1], r.childs_[2])
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
	CmnPNode
	CmnRule
	typ parenrType
}

func (r *ParenRule) isTerminal() bool {
	return false
}

func (r *ParenRule) id() string {
	s := ""
	if r.typ != parenrNone {
		s = string(r.typ)
	}
	return fmt.Sprintf("(%v)%v", r.onlyChild().id(), s)
}
func (r *ParenRule) String() string {
	return r.id()
}

//----------

// (0 childs, or temporarily 1 child that is a refrule)
type StringRule struct {
	CmnPNode
	CmnRule
	runes []rune
	typ   stringrType
}

func (r *StringRule) isTerminal() bool {
	return true
}
func (r *StringRule) id() string {
	s := ""
	if r.typ != stringrAnd {
		s = string(r.typ)
	}
	return fmt.Sprintf("%v%q", s, string(r.runes))
}
func (r *StringRule) String() string {
	return r.id()
}

//----------

// processor rule: allows processing rules at compile time. Ex: string operations, escape rune sequence (can fail and recover).
// (1 childs)
type ProcRule struct {
	CmnPNode
	CmnRule
	name string
}

func (r *ProcRule) isTerminal() bool {
	return false
}
func (r *ProcRule) id() string {
	return fmt.Sprintf("%v(%v)", r.name, r.onlyChild())
}
func (r *ProcRule) String() string {
	return r.id()
}

//----------

// (0 childs)
type FuncRule struct {
	CmnRule
	name string
	fn   PStateParseFn
}

func (r *FuncRule) isTerminal() bool {
	return true
}
func (r *FuncRule) id() string {
	return r.name
}
func (r *FuncRule) String() string {
	return r.id()
}

//----------

// (0 childs)
type SingletonRule struct {
	CmnPNode
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

//----------

// setup to be available in the grammars at ruleindex.go
var endRule = newSingletonRule("$", true)
var nilRule = newSingletonRule("nil", true)
var anyruneRule = newSingletonRule("anyrune", true)

// special start rule to know start/end (not a terminal)
var startRule = newSingletonRule("^^^", false)

//----------
//----------
//----------

// parenthesis rule type
type parenrType rune

const (
	parenrNone       parenrType = 0
	parenrOptional   parenrType = '?'
	parenrZeroOrMore parenrType = '*'
	parenrOneOrMore  parenrType = '+'
)

//----------

// string rule type
type stringrType rune

const (
	stringrAnd stringrType = 0   // sequence: "and" (default)
	stringrMid stringrType = '~' // sequence: middle match
	stringrOr  stringrType = '%' // individual runes
	stringrNot stringrType = '!' // individual runes: not
)

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

func ruleInnerStringRule(r Rule, upperType stringrType) (*StringRule, bool) {
	acceptType := func(typ2 stringrType) bool {
		switch upperType {
		case stringrAnd:
			switch typ2 {
			case stringrAnd:
				return true
			}
		case stringrNot:
			switch typ2 {
			case stringrOr:
				return true
			}
		case stringrMid:
			switch typ2 {
			case stringrAnd:
				return true
			}
		case stringrOr:
			switch typ2 {
			case stringrAnd, stringrOr:
				return true
			}
		}
		return false
	}

	switch t := r.(type) {
	case *StringRule:
		if acceptType(t.typ) {
			return t, true
		}
	case *DefRule:
		return ruleInnerStringRule(t.onlyChild(), upperType)
	case *OrRule:
		// concat "or" rules
		sr2 := &StringRule{typ: stringrOr}
		if acceptType(sr2.typ) {
			for _, c := range t.childs() {
				sr3, ok := ruleInnerStringRule(c, sr2.typ)
				if !ok {
					return nil, false
				}
				sr2.runes = append(sr2.runes, sr3.runes...)
			}
			return sr2, true
		}
	case *AndRule:
		// concat "and" rules
		sr2 := &StringRule{typ: stringrAnd}
		if acceptType(sr2.typ) {
			for _, c := range t.childs() {
				sr3, ok := ruleInnerStringRule(c, sr2.typ)
				if !ok {
					return nil, false
				}
				sr2.runes = append(sr2.runes, sr3.runes...)
			}
			return sr2, true
		}
	}
	return nil, false
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
