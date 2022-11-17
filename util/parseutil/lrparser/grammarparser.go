package lrparser

import (
	"errors"
	"fmt"
	"strconv"
	"unicode"

	"github.com/jmigpin/editor/util/parseutil"
)

type grammarParser struct {
	ri *RuleIndex
}

func newGrammarParser() *grammarParser {
	gp := &grammarParser{}
	gp.ri = newRuleIndex()
	return gp
}
func (gp *grammarParser) parse(fset *FileSet) (*RuleIndex, error) {
	ps := parseutil.NewPState(fset.Src)
	err := gp.parse2(ps)
	if err != nil {
		return nil, fset.Error2(err, ps.Pos)
	}
	return gp.ri, nil
}
func (gp *grammarParser) parse2(ps *PState) error {
	for {
		ok, err := gp.parse3(ps)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}
}
func (gp *grammarParser) parse3(ps *PState) (bool, error) {
	gp.parseOptionalSpacesOrComments(ps)
	if err := ps.MatchEof(); err == nil {
		return false, nil
	}
	if err := gp.parseRule(ps); err != nil {
		return false, err
	}
	return true, nil
}
func (gp *grammarParser) parseRule(ps *PState) error {
	i0 := ps.Pos

	// is start
	isStart := false
	err := ps.MatchString(defRuleStartSym)
	if err == nil {
		isStart = true
	}

	// rule name
	name, err := gp.parseName(ps)
	if err != nil {
		return err
	}
	if gp.ri.has(name) {
		return fmt.Errorf("rule already defined: %v", name)
	}

	gp.parseOptionalSpacesOrComments(ps)

	if err := ps.MatchRune('='); err != nil {
		return errors.New("expecting =")
	}

	gp.parseOptionalSpacesOrComments(ps)

	if err := gp.parseItemRule(ps); err != nil {
		return err
	}

	gp.parseOptionalSpacesOrComments(ps)

	if err := ps.MatchRune('.'); err != nil {
		return errors.New("expecting dot \".\" to close rule")
	}

	// setup
	dr := &DefRule{name: name, isStart: isStart}
	dr.setOnlyChild(ps.Node.(Rule))
	dr.setPos(i0, ps.Pos)
	gp.ri.set(dr.name, dr)

	return nil
}
func (gp *grammarParser) parseName(ps *PState) (string, error) {
	name := ""
	for {
		ps2 := ps.Copy()
		ru, err := ps.ReadRune()
		if err != nil {
			break
		}

		// special case: accept endrule
		if name+string(ru) == endRule.id() {
			name += string(ru)
			break
		}

		if !(unicode.IsLetter(ru) || unicode.IsDigit(ru) || ru == '_') {
			ps.Set(ps2) // go back
			break
		}
		name += string(ru)
	}
	if name == "" {
		return "", fmt.Errorf("empty name")
	}
	return name, nil
}

//----------

func (gp *grammarParser) parseItemRule(ps *PState) error {
	return gp.parseIfRule(ps) // precedence tree construction
}

//----------

func (gp *grammarParser) parseIfRule(ps *PState) error {
	i0 := ps.Pos

	if err := ps.MatchString("if "); err != nil {
		return gp.parseOrRule(ps) // ok, not an ifrule
	}

	_, ok := gp.parseRefRule(ps)
	if !ok {
		return fmt.Errorf("expecting name")
	}
	nameRef := ps.Node.(Rule)

	// then
	gp.parseOptionalSpacesOrComments(ps)
	if err := ps.MatchString("?"); err != nil {
		return fmt.Errorf("expecting '?'")
	}
	gp.parseOptionalSpacesOrComments(ps)
	if err := gp.parseItemRule(ps); err != nil {
		return err
	}
	thenRule := ps.Node.(Rule)

	// else
	gp.parseOptionalSpacesOrComments(ps)
	if err := ps.MatchString(":"); err != nil {
		return fmt.Errorf("expecting ':'")
	}
	gp.parseOptionalSpacesOrComments(ps)
	if err := gp.parseItemRule(ps); err != nil {
		return err
	}
	elseRule := ps.Node.(Rule)

	// setup
	res := &IfRule{}
	res.addChild(nameRef)
	res.addChild(thenRule)
	res.addChild(elseRule)
	res.setPos(i0, ps.Pos)
	ps.Node = res

	return nil
}

//----------

func (gp *grammarParser) parseOrRule(ps *PState) error {
	i0 := ps.Pos
	ps2 := ps.Copy()
	w := []Rule{}
	for i := 0; ; i++ {
		// handle separator
		if i > 0 {
			gp.parseOptionalSpacesOrComments(ps2)
			ps3 := ps2.Copy()
			if err := ps2.MatchRune('|'); err != nil {
				if i == 1 {
					ps.Set(ps3)
					return nil // ok, just not an OR
				}

				res := &OrRule{}
				res.childs_ = w
				res.setPos(i0, ps2.Pos)
				ps2.Node = res

				ps.Set(ps2)
				return nil // ok
			}

			gp.parseOptionalSpacesOrComments(ps2)
		}

		if err := gp.parseAndRule(ps2); err != nil {
			//if i == 0 {
			//	ps.set(ps2) // better error?
			//	return err  // fail, no rule
			//}
			ps.Set(ps2)
			return err // fail, not expecting error after sep
		}

		resRule := ps2.Node.(Rule)
		w = append(w, resRule)
	}
}
func (gp *grammarParser) parseAndRule(ps *PState) error {
	ps2 := ps.Copy()
	w := []Rule{}
	for i := 0; ; i++ {
		// handle separator
		if i > 0 {
			gp.parseOptionalSpacesOrComments(ps2)
		}

		if err := gp.parseBasicItemRule(ps2); err != nil {
			if i == 0 {
				ps.Set(ps2) // better error?
				return err  // fail, no rule
			}
			if i == 1 {
				ps.Set(ps2)
				return nil // ok, just not an AND
			}
			break // ok, don't include the spaces
		}

		resRule := ps2.Node.(Rule)
		w = append(w, resRule)
	}

	res := &AndRule{}
	res.childs_ = w
	res.setPos(ps.Pos, ps2.Pos)
	ps2.Node = res
	ps.Set(ps2)

	return nil
}

//----------

func (gp *grammarParser) parseBasicItemRule(ps *PState) error {
	if err, ok := gp.parseProcRule(ps); ok {
		return err
	}
	if err, ok := gp.parseRefRule(ps); ok {
		return err
	}
	if err, ok := gp.parseStringRule(ps); ok {
		return err
	}
	if err, ok := gp.parseParenRule(ps); ok {
		return err
	}
	return errors.New("unable to parse basic item")
}
func (gp *grammarParser) parseProcRule(ps *PState) (error, bool) {
	i0 := ps.Pos
	// header
	callRuleSym := "&"
	if err := ps.MatchString(callRuleSym); err != nil {
		return err, false
	}
	// name
	name, err := gp.parseName(ps)
	if err != nil {
		return err, true
	}
	// arg
	if err := ps.MatchRune('('); err != nil {
		return err, true
	}
	if err := gp.parseItemRule(ps); err != nil {
		return err, true
	}
	ruleX := ps.Node.(Rule)
	if err := ps.MatchRune(')'); err != nil {
		return err, true
	}

	res := &ProcRule{}
	res.name = callRuleSym + name
	res.addChild(ruleX)
	res.setPos(i0, ps.Pos)
	ps.Node = res

	return nil, true
}
func (gp *grammarParser) parseRefRule(ps *PState) (error, bool) {
	i0 := ps.Pos

	// options
	ps2 := ps.Copy()
	stype, haveStringrType := gp.parseStringRuleType(ps2)

	name, err := gp.parseName(ps2)
	if err != nil {
		return nil, false // err is lost
	}

	ps.Set(ps2) // advance

	res := &RefRule{name: name}
	res.setPos(i0, ps.Pos)
	ps.Node = res
	if haveStringrType {
		sr := &StringRule{}
		sr.typ = stype
		sr.addChild(res)
		ps.Node = sr
	}

	return nil, true
}

func (gp *grammarParser) parseStringRule(ps *PState) (error, bool) {
	i0 := ps.Pos

	// options
	ps2 := ps.Copy()
	st, _ := gp.parseStringRuleType(ps2)

	quoteRu, err := ps2.ReadRune()
	if err != nil {
		return err, false
	}
	if quoteRu != '"' {
		return errors.New("expecting quote"), false
	}

	ps.Set(ps2) // advance

	s := string(quoteRu)
	esc := '\\' // alows to escape the quote
	escaping := false
	for {
		ru, err := ps.ReadRune()
		if err != nil {
			break
		}
		s += string(ru)
		if escaping {
			escaping = false
			continue
		} else if ru == esc {
			escaping = true
		}
		if ru == quoteRu {
			break
		}
	}

	// needed: ex: transforms "\n" (2 runes) into a single '\n'
	u, err := strconv.Unquote(s)
	if err != nil {
		return err, true
	}

	res := &StringRule{runes: []rune(u), typ: st}
	res.setPos(i0, ps.Pos)
	ps.Node = res
	return nil, true
}
func (gp *grammarParser) parseStringRuleType(ps *PState) (stringrType, bool) {
	//  options
	ps2 := ps.Copy()
	ru, err := ps2.ReadRune()
	if err != nil {
		return stringrAnd, false
	}
	switch t := stringrType(ru); t {
	case stringrOr,
		stringrMid,
		stringrNot:
		ps.Set(ps2) // advance
		return t, true
	case stringrAnd:
		return stringrAnd, true // default
	default:
		return stringrAnd, false
	}
}
func (gp *grammarParser) parseParenRule(ps *PState) (error, bool) {
	i0 := ps.Pos
	if err := ps.MatchRune('('); err != nil {
		return err, false
	}
	if err := gp.parseItemRule(ps); err != nil {
		return err, true
	}
	ruleX := ps.Node.(Rule)
	if err := ps.MatchRune(')'); err != nil {
		return err, true
	}

	// options
	ps2 := ps.Copy()
	ru, err := ps2.ReadRune()
	if err != nil {
		ru = 0
	}
	pt := parenrNone
	switch t := parenrType(ru); t {
	case parenrNone,
		parenrOptional,
		parenrZeroOrMore,
		parenrOneOrMore:
		pt = t
		ps.Set(ps2) // advance
	}

	//if pt == parenrZeroOrMore {
	//	u := newParenZeroOrMoreRule(ruleX)
	//	u.setPos(i0, ps.Pos)
	//	ps.Node = u
	//} else {
	u := &ParenRule{typ: pt}
	u.setOnlyChild(ruleX)
	u.setPos(i0, ps.Pos)
	ps.Node = u
	//}

	return nil, true
}

//----------

func (gp *grammarParser) parseOptionalSpacesOrComments(ps *PState) {
	for {
		if ps.ConsumeSpacesIncludingNL() {
			continue
		}
		if gp.parseComments(ps) {
			continue
		}
		break
	}
}

//func (gp *grammarParser) parseEmptyLine(ps *PState) bool {
//	if err := ps.MatchRune('\n'); err == nil {
//		return true
//	}
//	return false
//}

func (gp *grammarParser) parseComments(ps *PState) bool {
	if err := ps.MatchRune('#'); err == nil {
		_ = ps.ConsumeToNLIncluding()
		return true
	}
	if err := ps.MatchString("//"); err == nil {
		_ = ps.ConsumeToNLIncluding()
		return true
	}
	return false
}
