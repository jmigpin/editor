package lrparser

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/jmigpin/editor/util/goutil"
)

type Rule interface {
	id() string
	iterChilds(fn func(index int, ref *Rule) error) error
	isTerminal() bool
	String() string

	// TODO: consider
	// parse(*PState) error // for terminal rules
}

//----------

// common rule
type CmnRule struct {
	childs []Rule
}

func (r *CmnRule) addChilds(r2 Rule) {
	r.childs = append(r.childs, r2)
}
func (r *CmnRule) onlyChild() Rule {
	return r.childs[0]
}
func (r *CmnRule) setOnlyChild(r2 Rule) {
	r.childs = r.childs[:0]
	r.addChilds(r2)
}
func (r *CmnRule) iterChilds(fn func(index int, ref *Rule) error) error {
	for i := 0; i < len(r.childs); i++ {
		if err := fn(i, &r.childs[i]); err != nil {
			return err
		}
	}
	return nil
}

//----------
//----------
//----------

// definition rule
type DefRule struct {
	CmnPNode
	CmnRule
	name       string
	declId     int  // declaration order, 0=inserted, >=1=declared
	isStart    bool // has "start" symbol in the grammar
	isLoop     bool
	ifRuleName string // name of conditional rule to make this rule work
}

func (r *DefRule) isTerminal() bool {
	return false
}
func (r *DefRule) id() string {
	return r.name
}
func (r *DefRule) String() string {
	s := r.id()
	if r.isStart {
		s = defRuleStartSym + s
	}
	return fmt.Sprintf("%v = %v", s, r.onlyChild().id())
}

var defRuleStartSym = "^" // used in grammar

//----------

type RefRule struct { // reference to a rule
	CmnPNode
	CmnRule
	name string
}

func (r *RefRule) isTerminal() bool {
	return false
}
func (r *RefRule) id() string {
	return fmt.Sprintf("{ref:%v}", r.name)
}
func (r *RefRule) String() string {
	return r.id()
}

//----------

type AndRule struct {
	CmnPNode
	CmnRule
}

func (r *AndRule) isTerminal() bool {
	return false
}
func (r *AndRule) id() string {
	w := []string{}
	for _, r := range r.childs {
		w = append(w, r.id())
	}
	return strings.Join(w, " ")
}
func (r *AndRule) String() string {
	return r.id()
}

//----------

type OrRule struct {
	CmnPNode
	CmnRule
}

func (r *OrRule) isTerminal() bool {
	return false
}
func (r *OrRule) id() string {
	w := []string{}
	for _, r := range r.childs {
		w = append(w, r.id())
	}
	return strings.Join(w, " | ")
}
func (r *OrRule) String() string {
	return r.id()
}

//----------

type ParenRule struct { // parenthesis, ex: (aaa (bbb|ccc))
	CmnPNode
	CmnRule
}

func (r *ParenRule) isTerminal() bool {
	return false
}
func (r *ParenRule) id() string {
	return fmt.Sprintf("(%v)", r.onlyChild().id())
}
func (r *ParenRule) String() string {
	return r.id()
}

//----------

type ParenOptionalRule struct {
	ParenRule
}

func (r *ParenOptionalRule) id() string {
	return fmt.Sprintf("(%v)?", r.onlyChild().id())
}
func (r *ParenOptionalRule) String() string {
	return r.id()
}

//----------

type ParenZeroOrMoreRule struct {
	ParenRule
}

func (r *ParenZeroOrMoreRule) id() string {
	return fmt.Sprintf("(%v)*", r.onlyChild().id())
}
func (r *ParenZeroOrMoreRule) String() string {
	return r.id()
}

//----------

type ParenOneOrMoreRule struct {
	ParenRule
}

func (r *ParenOneOrMoreRule) id() string {
	return fmt.Sprintf("(%v)+", r.onlyChild().id())
}
func (r *ParenOneOrMoreRule) String() string {
	return r.id()
}

//----------

type StringRule struct {
	CmnPNode
	CmnRule
	runes []rune
}

func (r *StringRule) isTerminal() bool {
	return true
}
func (r *StringRule) id() string {
	u := string(r.runes)
	//u = strings.ReplaceAll(u, "%", "%%")
	return fmt.Sprintf("%q", u)
}
func (r *StringRule) String() string {
	return r.id()
}

//----------

// individual rune match
type StringOrRule struct {
	StringRule
}

func (r *StringOrRule) id() string {
	return fmt.Sprintf("%v&", r.StringRule.id())
}
func (r *StringOrRule) String() string {
	return r.id()
}

//----------

// middle match
type StringMidRule struct {
	StringRule
}

func (r *StringMidRule) id() string {
	return fmt.Sprintf("%v~", r.StringRule.id())
}
func (r *StringMidRule) String() string {
	return r.id()
}

//----------

type FuncRule struct {
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

func ruleSequence(r Rule) []Rule {
	switch t := r.(type) {
	case *AndRule:
		return t.childs
	default:
		return []Rule{t}
	}
}
func ruleLen(r Rule) int {
	return len(ruleSequence(r))
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
		for _, r2 := range t.childs {
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

func ruleProductions(r Rule) ([]Rule, bool) {
	// NOTE: can't create new rules here because they won't be unique (new mem address) and fail as a map index (used in the rest of the code; will allow endless loops)

	switch t := r.(type) {
	case *DefRule:
		return ruleProductions2(t.onlyChild()), true
	default:
		if r.isTerminal() {
			return nil, false
		}
		panic(goutil.TodoErrorType(t))
	}
}
func ruleProductions2(r Rule) []Rule {
	switch t := r.(type) {
	case *OrRule:
		return t.childs
	default:
		return []Rule{t}
	}
}

//----------

func walkRuleChilds(rule Rule, fn func(*Rule) error) error {
	return rule.iterChilds(func(index int, ref *Rule) error {
		return fn(ref)
	})
}
