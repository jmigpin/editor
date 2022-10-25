package lrparser

import (
	"errors"
	"fmt"
	"strconv"
	"unicode"
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
	ps := &PState{src: fset.Src}
	err := gp.parse2(ps)
	if err != nil {
		return nil, fset.Error2(err, ps.i)
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
	if err := gp.parseRule(ps); err == nil {
		return true, err
	}
	if err := ps.matchEof(); err == nil {
		return false, nil
	}
	return false, errors.New("unexpected")
}
func (gp *grammarParser) parseRule(ps *PState) error {
	i0 := ps.i

	// is start
	isStart := false
	err := ps.matchString(defRuleStartSym)
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
		return errors.New("expecting .")
	}

	// setup
	dr := &DefRule{name: name, isStart: isStart}
	dr.setOnlyChild(ps.parseNode.(Rule))
	dr.setPos(i0, ps.i)
	gp.ri.set(dr.name, dr)

	return nil
}
func (gp *grammarParser) parseName(ps *PState) (string, error) {
	name := ""
	for {
		ps2 := ps.copy()
		ru, err := ps.readRune()
		if err != nil {
			break
		}
		if !(unicode.IsLetter(ru) || unicode.IsDigit(ru) || ru == '_') {
			ps.set(ps2) // go back
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
	i0 := ps.i

	if err := ps.matchString("if "); err != nil {
		return gp.parseOrRule(ps) // ok, not an ifrule
	}

	_, ok := gp.parseRefRule(ps)
	if !ok {
		return fmt.Errorf("expecting name")
	}
	nameRef := ps.parseNode.(Rule)

	// then
	gp.parseOptionalSpacesOrComments(ps)
	if err := ps.matchString("?"); err != nil {
		return fmt.Errorf("expecting '?'")
	}
	gp.parseOptionalSpacesOrComments(ps)
	if err := gp.parseItemRule(ps); err != nil {
		return err
	}
	thenRule := ps.parseNode.(Rule)

	// else
	gp.parseOptionalSpacesOrComments(ps)
	if err := ps.matchString(":"); err != nil {
		return fmt.Errorf("expecting ':'")
	}
	gp.parseOptionalSpacesOrComments(ps)
	if err := gp.parseItemRule(ps); err != nil {
		return err
	}
	elseRule := ps.parseNode.(Rule)

	// setup
	res := &IfRule{}
	res.addChild(nameRef)
	res.addChild(thenRule)
	res.addChild(elseRule)
	res.setPos(i0, ps.i)
	ps.parseNode = res

	return nil
}

//----------

func (gp *grammarParser) parseOrRule(ps *PState) error {
	i0 := ps.i
	ps2 := ps.copy()
	w := []Rule{}
	for i := 0; ; i++ {
		// handle separator
		if i > 0 {
			gp.parseOptionalSpacesOrComments(ps2)
			ps3 := ps2.copy()
			if err := ps2.MatchRune('|'); err != nil {
				if i == 1 {
					ps.set(ps3)
					return nil // ok, just not an OR
				}

				res := &OrRule{}
				res.childs2 = w
				res.setPos(i0, ps2.i)
				ps2.parseNode = res

				ps.set(ps2)
				return nil // ok
			}

			gp.parseOptionalSpacesOrComments(ps2)
		}

		if err := gp.parseAndRule(ps2); err != nil {
			if i == 0 {
				ps.set(ps2) // better error?
				return err  // fail, no rule
			}
			ps.set(ps2)
			return err // fail, not expecting error after sep
		}

		resRule := ps2.parseNode.(Rule)
		w = append(w, resRule)
	}
}
func (gp *grammarParser) parseAndRule(ps *PState) error {
	ps2 := ps.copy()
	w := []Rule{}
	for i := 0; ; i++ {
		// handle separator
		if i > 0 {
			gp.parseOptionalSpacesOrComments(ps2)
		}

		if err := gp.parseBasicItemRule(ps2); err != nil {
			if i == 0 {
				ps.set(ps2) // better error?
				return err  // fail, no rule
			}
			if i == 1 {
				ps.set(ps2)
				return nil // ok, just not an AND
			}
			break // ok, don't include the spaces
		}

		resRule := ps2.parseNode.(Rule)
		w = append(w, resRule)
	}

	res := &AndRule{}
	res.childs2 = w
	res.setPos(ps.i, ps2.i)
	ps2.parseNode = res
	ps.set(ps2)

	return nil
}

//----------

func (gp *grammarParser) parseBasicItemRule(ps *PState) error {
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
func (gp *grammarParser) parseRefRule(ps *PState) (error, bool) {
	i0 := ps.i
	name, err := gp.parseName(ps)
	if err != nil {
		return nil, false // err is lost
	}

	// options
	st, _ := gp.parseStringRuleType(ps)

	res := &RefRule{name: name, stringrType: st}
	res.setPos(i0, ps.i)
	ps.parseNode = res
	return nil, true
}

func (gp *grammarParser) parseStringRule(ps *PState) (error, bool) {
	i0 := ps.i
	ps2 := ps.copy()
	quoteRu, err := ps.readRune()
	if err != nil {
		return err, false
	}
	if quoteRu != '"' {
		ps.set(ps2) // go back
		return errors.New("expecting quote"), false
	}

	s := string(quoteRu)
	esc := '\\' // alows to escape the quote
	escaping := false
	for {
		ru, err := ps.readRune()
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

	// options
	st, _ := gp.parseStringRuleType(ps)

	res := &StringRule{runes: []rune(u), typ: st}
	res.setPos(i0, ps.i)
	ps.parseNode = res
	return nil, true
}
func (gp *grammarParser) parseStringRuleType(ps *PState) (stringrType, bool) {
	//  options
	ps2 := ps.copy()
	ru, err := ps2.readRune()
	if err != nil {
		ru = 0
	}
	switch t := stringrType(ru); t {
	case stringrNone,
		stringrRunes,
		stringrMidMatch:
		ps.set(ps2) // advance
		return t, true
	}
	return stringrNone, false
}
func (gp *grammarParser) parseParenRule(ps *PState) (error, bool) {
	i0 := ps.i
	if err := ps.MatchRune('('); err != nil {
		return err, false
	}
	if err := gp.parseItemRule(ps); err != nil {
		return err, true
	}
	ruleX := ps.parseNode.(Rule)
	if err := ps.MatchRune(')'); err != nil {
		return err, true
	}

	//  options
	ps2 := ps.copy()
	ru, err := ps2.readRune()
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
		ps.set(ps2) // advance
	}

	u := &ParenRule{typ: pt}
	u.setOnlyChild(ruleX)
	u.setPos(i0, ps.i)
	ps.parseNode = u

	return nil, true
}

//----------

func (gp *grammarParser) parseOptionalSpacesOrComments(ps *PState) {
	for {
		if ps.consumeSpacesIncludingNL() {
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
		_ = ps.consumeToNLIncluding()
		return true
	}
	if err := ps.matchString("//"); err == nil {
		_ = ps.consumeToNLIncluding()
		return true
	}
	return false
}
