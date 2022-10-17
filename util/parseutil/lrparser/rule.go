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
	//productions() []Rule // or rule
	//sequence() []Rule    // and rule

	childs() []Rule // productions
	iterChildRefs(fn func(index int, ref *Rule) error) error

	String() string

	// TODO: consider
	// parse(*PState) error // for terminal rules

	// TODO: cmnrule0childs
	// TODO: cmnrule1child
	// TODO: cmnruleNchilds
}

//----------
//----------
//----------

// common rule
type CmnRule struct {
	childs2 []Rule
}

//----------

func (r *CmnRule) addChild(r2 Rule) {
	r.childs2 = append(r.childs2, r2)
}
func (r *CmnRule) onlyChild() Rule {
	return r.childs2[0]
}
func (r *CmnRule) setOnlyChild(r2 Rule) {
	r.childs2 = r.childs2[:0]
	r.addChild(r2)
}

//----------

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
type DefRule struct { // (1 child)
	CmnPNode
	CmnRule
	name    string
	declId  int  // declaration order, 0=inserted, >=1=declared
	isStart bool // has "start" symbol in the grammar
	isLoop  bool
}

func (r *DefRule) isTerminal() bool {
	return false
}
func (r *DefRule) id() string {
	s := ""
	if r.isStart {
		s += defRuleStartSym
	}
	if r.isLoop {
		s += "l"
	}
	if s != "" {
		s = ":" + s
	}
	return fmt.Sprintf("{d%v:%v}", s, r.name)
	//return r.name
}
func (r *DefRule) String() string {
	return fmt.Sprintf("%v = %v", r.id(), r.onlyChild().id())
}

var defRuleStartSym = "^" // used in grammar

//----------

// reference to a rule
// replaced in dereference phase
type RefRule struct { // (0 childs)
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

type AndRule struct { // (n childs)
	CmnPNode
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
	//return strings.Join(w, " ")
}
func (r *AndRule) String() string {
	return r.id()
}

//----------

type OrRule struct { // (n childs)
	CmnPNode
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
	u := strings.Join(w, " | ")
	return fmt.Sprintf("[%v]", u)
	//return strings.Join(w, " | ")
}
func (r *OrRule) String() string {
	return r.id()
}

//----------

// replaced in dereference phase
type IfRule struct { // (3 childs: [conditional,then,else])
	CmnPNode
	CmnRule
}

func (r *IfRule) isTerminal() bool {
	return false
}
func (r *IfRule) id() string {
	return fmt.Sprintf("{if %v ? %v : %v}", r.childs2[0], r.childs2[1], r.childs2[2])
}
func (r *IfRule) String() string {
	return r.id()
}

//----------

// To be used in src code and then found by IfRule; the value is observed when building the contentparser, not at parse time
type BoolRule struct { // (0 childs)
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
type ParenRule struct { // (1 child)
	CmnPNode
	CmnRule
	typ parenType
}

func (r *ParenRule) isTerminal() bool {
	return false
}
func (r *ParenRule) id() string {
	s := ""
	if r.typ != 0 {
		s = string(r.typ)
	}
	return fmt.Sprintf("(%v)%v", r.onlyChild().id(), s)
}
func (r *ParenRule) String() string {
	return r.id()
}

//----------

type StringRule struct { // (0 childs)
	CmnPNode
	CmnRule
	runes []rune
	typ   parenType
}

func (r *StringRule) isTerminal() bool {
	return true
}
func (r *StringRule) id() string {
	s := ""
	if r.typ != 0 {
		s = string(r.typ)
	}
	return fmt.Sprintf("%q%v", string(r.runes), s)
}
func (r *StringRule) String() string {
	return r.id()
}

//----------

type FuncRule struct { // (0 childs)
	CmnRule
	name string
	fn   pstateParseFn
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

type SingletonRule struct { // (0 childs)
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

type parenType rune

const (
	parenNone           parenType = 0 // parenrule/stringrule
	parenOptional       parenType = '?'
	parenZeroOrMore     parenType = '*'
	parenOneOrMore      parenType = '+'
	parenStringRunes    parenType = '&' // parenrule/stringrule
	parenStringMidMatch parenType = '~' // parenrule/stringrule
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

func ruleVDProductions(r Rule) []Rule {
	return ruleFirstProductions(r)
}
func ruleFirstProductions(r Rule) []Rule {
	switch t := r.(type) {
	case *AndRule: // andrule childs are not productions
		return []Rule{t}
		//case *DefRule:
		//	switch t2 := t.onlyChild().(type) {
		//	case *OrRule:
		//		return t2.childs()
		//	}
	}
	return r.childs()
}
func ruleFirstSequence(r Rule) []Rule {
	switch t := r.(type) {
	case *AndRule: // andrule is the only rule whose childs provide a sequence
		return t.childs()
	default:
		return []Rule{t}
	}
}

func ruleIsLoop(r Rule) bool {
	dr, ok := r.(*DefRule)
	return ok && dr.isLoop
}
func ruleCanBeNil(r Rule) bool {
	if r == nilRule {
		return true
	}
	switch t := r.(type) {
	case *DefRule:
		return ruleCanBeNil(t.onlyChild())
	case *OrRule:
		for _, r2 := range t.childs2 {
			if ruleCanBeNil(r2) {
				return true
			}
		}
	}
	return false
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
