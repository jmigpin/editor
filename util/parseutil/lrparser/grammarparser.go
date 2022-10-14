package lrparser

import (
	"errors"
	"fmt"
	"strconv"
	"unicode"
)

type grammarParser struct {
	ri     *RuleIndex
	declId int
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
		ok, err := gp.parseLine(ps)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}
}
func (gp *grammarParser) parseLine(ps *PState) (bool, error) {
	_ = ps.consumeSpacesExcludingNL() // optional
	// empty lines
	if err := ps.MatchRune('\n'); err == nil {
		return true, nil
	}
	// comments
	if err := ps.MatchRune('#'); err == nil {
		_ = ps.consumeToNLIncluding()
		return true, nil
	}
	if err := ps.matchString("//"); err == nil {
		_ = ps.consumeToNLIncluding()
		return true, nil
	}
	// rule
	if err := ps.matchString("rule "); err == nil {
		if err := gp.parseRule(ps, ""); err != nil {
			return false, err
		}
		return true, nil
	}
	// if bool rule
	if err := ps.matchString("if "); err == nil {
		if err := gp.parseIfRule(ps); err != nil {
			return false, err
		}
		return true, nil
	}
	// eof
	if err := ps.matchEof(); err == nil {
		return false, nil
	}
	return false, errors.New("unexpected line")
}
func (gp *grammarParser) parseRule(ps *PState, ifRuleName string) error {
	_ = ps.consumeSpacesExcludingNL() // optional

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

	_ = ps.consumeSpacesExcludingNL() // optional

	if err := ps.MatchRune('='); err != nil {
		return errors.New("expecting =")
	}

	_ = ps.consumeSpacesExcludingNL2() // optional

	if err := gp.parseItemRule(ps); err != nil {
		return err
	}

	// setup defrule
	dr := &DefRule{name: name, ifRuleName: ifRuleName, isStart: isStart}
	dr.setOnlyChild(ps.parseNode.(Rule))
	gp.declId++
	dr.declId = gp.declId
	dr.setPos(i0, ps.i)
	gp.ri.set(dr.name, dr)

	return nil
}

//func (gp *grammarParser) parseRuleNameAndEqual(ps *PState) (string, error) {
//	name := ""
//	for {
//		ps2 := ps.copy()
//		ru, err := ps.readRune()
//		if err != nil {
//			break
//		}
//		if !(unicode.IsLetter(ru) || unicode.IsDigit(ru) || ru == '_') {
//			ps.set(ps2)
//			break
//		}
//		name += string(ru)
//	}
//	if name == "" {
//		return "", errors.New("expecting name")
//	}

//	if gp.ri.has(name) {
//		return "", fmt.Errorf("rule already defined: %v", name)
//	}

//	_ = ps.consumeSpacesExcludingNL() // optional

//	if err := ps.MatchRune('='); err != nil {
//		return "", errors.New("expecting =")
//	}

//	_ = ps.consumeSpacesExcludingNL2() // optional

//		return name, nil
//	}
func (gp *grammarParser) parseName(ps *PState) (string, error) {
	name := ""
	for {
		ps2 := ps.copy()
		ru, err := ps.readRune()
		if err != nil {
			break
		}
		if !(unicode.IsLetter(ru) || unicode.IsDigit(ru) || ru == '_') {
			ps.set(ps2)
			break
		}
		name += string(ru)
	}
	if name == "" {
		return "", errors.New("expecting name")
	}
	return name, nil
}

//----------

func (gp *grammarParser) parseIfRule(ps *PState) error {
	_ = ps.consumeSpacesExcludingNL() // optional

	// bool rule name
	name, err := gp.parseName(ps)
	if err != nil {
		return err
	}

	if !ps.consumeSpacesExcludingNL() {
		return fmt.Errorf("expecting space")
	}

	if err := ps.matchString("rule "); err != nil {
		return err
	}
	if err := gp.parseRule(ps, name); err != nil {
		return err
	}
	return nil
}

//----------

func (gp *grammarParser) parseItemRule(ps *PState) error {
	return gp.parseOrRule(ps) // precedence tree construction
}

//----------

func (gp *grammarParser) parseOrRule(ps *PState) error {
	i0 := ps.i
	ps2 := ps.copy()
	w := []Rule{}
	for i := 0; ; i++ {
		// handle separator
		if i > 0 {
			_ = ps2.consumeSpacesExcludingNL() // optional
			ps3 := ps2.copy()
			if err := ps2.MatchRune('|'); err != nil {
				if i == 1 {
					ps.set(ps3)
					return nil // ok, just not an OR
				}

				res := &OrRule{}
				res.childs = w
				res.setPos(i0, ps2.i)
				ps2.parseNode = res

				ps.set(ps2)
				return nil // ok
			}

			//_ = ps2.consumeSpacesExcludingNL() // optional
			_ = ps2.consumeSpacesIncludingNL() // optional
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
			ok := ps2.consumeSpacesExcludingNL2() // optional
			if !ok {
				if i == 1 {
					ps.set(ps2)
					return nil // ok, just not an AND
				}
				break // ok
			}
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
	res.childs = w
	res.setPos(ps.i, ps2.i)
	ps2.parseNode = res
	ps.set(ps2)

	return nil
}

//----------

func (gp *grammarParser) parseBasicItemRule(ps *PState) error {
	if err, ok := gp.parseIdRule(ps); ok {
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

func (gp *grammarParser) parseIdRule(ps *PState) (error, bool) {
	i0 := ps.i
	s := ""
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
		s += string(ru)
	}
	if len(s) == 0 {
		return nil, false
	}

	res := &RefRule{name: s}
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

	// transform "\n" (2 runes) into a single '\n'
	u, err := strconv.Unquote(s)
	if err != nil {
		return err, true
	}

	res := &StringRule{runes: []rune(u)}
	res.setPos(i0, ps.i)
	ps.parseNode = res

	// extension
	ps3 := ps.copy()
	ru, err := ps3.readRune()
	if err == nil {
		switch ru {
		case '&':
			u := &StringOrRule{}
			u.StringRule = *res
			ps.parseNode = u
			ps.i = ps3.i //advance
		//case '!':
		//	u := &StringNotRule{}
		//	u.StringRule = *res
		//	ps.parseNode = u
		//	ps.i = ps3.i //advance
		case '~':
			u := &StringMidRule{}
			u.StringRule = *res
			ps.parseNode = u
			ps.i = ps3.i //advance
		}
	}
	return nil, true
}

func (gp *grammarParser) parseParenRule(ps *PState) (error, bool) {
	i0 := ps.i
	//_ = ps.consumeSpacesIncludingNL() // optional
	if err := ps.MatchRune('('); err != nil {
		return err, false
	}
	if err := gp.parseItemRule(ps); err != nil {
		return err, true
	}
	//_ = ps.consumeSpacesIncludingNL() // optional
	ruleX := ps.parseNode.(Rule)
	if err := ps.MatchRune(')'); err != nil {
		return err, true
	}

	//  options
	ps2 := ps.copy()
	ru, _ := ps2.readRune()
	switch ru {
	case '?':
		ps.set(ps2)
		u := &ParenOptionalRule{}
		u.setOnlyChild(ruleX)
		u.setPos(i0, ps.i)
		ps.parseNode = u
	case '*':
		ps.set(ps2)
		u := &ParenZeroOrMoreRule{}
		u.setOnlyChild(ruleX)
		u.setPos(i0, ps.i)
		ps.parseNode = u
	case '+':
		ps.set(ps2)
		u := &ParenOneOrMoreRule{}
		u.setOnlyChild(ruleX)
		u.setPos(i0, ps.i)
		ps.parseNode = u
	default:
		u := &ParenRule{}
		u.setOnlyChild(ruleX)
		u.setPos(i0, ps.i)
		ps.parseNode = u
	}
	return nil, true
}
