package lrparser

import (
	"errors"
	"fmt"
	"strconv"
)

type grammarParser struct {
	ri *RuleIndex
}

func newGrammarParser(ri *RuleIndex) *grammarParser {
	gp := &grammarParser{ri: ri}
	return gp
}
func (gp *grammarParser) parse(fset *FileSet) error {
	ps := NewPState(fset.Src)
	err := gp.parse2(ps)
	if err != nil {
		return fset.Error2(err, ps.Pos)
	}
	return nil
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
	if ps.M.Eof() {
		return false, nil
	}
	if err := gp.parseDefRule(ps); err != nil {
		return false, err
	}
	return true, nil
}
func (gp *grammarParser) parseDefRule(ps *PState) error {
	pos0 := ps.KeepPos()

	// options
	isStart := false
	if err := ps.M.Sequence(defRuleStartSym); err == nil {
		isStart = true
	}
	isNoPrint := false
	if err := ps.M.Sequence(defRuleNoPrintSym); err == nil {
		isNoPrint = true
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

	if err := ps.M.Rune('='); err != nil {
		return errors.New("expecting =")
	}

	gp.parseOptionalSpacesOrComments(ps)

	if err := gp.parseItemRule(ps); err != nil {
		return err
	}

	gp.parseOptionalSpacesOrComments(ps)

	if err := ps.M.Rune(';'); err != nil {
		return errors.New("expecting close rule \";\"?")
	}

	// setup
	dr := &DefRule{name: name, isStart: isStart, isNoPrint: isNoPrint}
	dr.setOnlyChild(ps.Node.(Rule))
	dr.SetPos(pos0.Pos, ps.Pos)
	gp.ri.set(dr.name, dr)
	ps.Node = dr

	return nil
}
func (gp *grammarParser) parseName(ps *PState) (string, error) {
	u := "[_a-zA-Z][_a-zA-Z0-9$]*"
	pos0 := ps.KeepPos()
	if err := ps.M.RegexpFromStartCached(u, 100); err != nil {
		return "", err
	}
	name := string(pos0.Bytes())
	return name, nil
}

//----------

func (gp *grammarParser) parseItemRule(ps *PState) error {
	// NOTE: taking into consideration precedence tree construction

	if err, ok := gp.parseIfRule(ps); ok {
		return err
	}
	return gp.parseOrRule(ps)
}

//----------

func (gp *grammarParser) parseIfRule(ps *PState) (error, bool) {
	if err := ps.M.Sequence("if "); err != nil {
		return err, false // not an ifrule
	}
	return gp.parseIfRule2(ps), true
}
func (gp *grammarParser) parseIfRule2(ps *PState) error {
	i0 := ps.Pos

	_, ok := gp.parseRefRule(ps)
	if !ok {
		return fmt.Errorf("expecting name")
	}
	nameRef := ps.Node.(Rule)

	// then
	gp.parseOptionalSpacesOrComments(ps)
	if err := ps.M.Rune('?'); err != nil {
		return fmt.Errorf("expecting '?'")
	}
	gp.parseOptionalSpacesOrComments(ps)
	if err := gp.parseItemRule(ps); err != nil {
		return err
	}
	thenRule := ps.Node.(Rule)

	// else
	gp.parseOptionalSpacesOrComments(ps)
	if err := ps.M.Rune(':'); err != nil {
		return fmt.Errorf("expecting ':'")
	}
	gp.parseOptionalSpacesOrComments(ps)
	if err := gp.parseItemRule(ps); err != nil {
		return err
	}
	elseRule := ps.Node.(Rule)

	// setup
	res := &IfRule{}
	res.addChilds(nameRef)
	res.addChilds(thenRule)
	res.addChilds(elseRule)
	res.SetPos(i0, ps.Pos)
	ps.Node = res

	return nil
}

//----------

func (gp *grammarParser) parseOrRule(ps *PState) error {
	pos0 := ps.KeepPos()
	w := []Rule{}
	for i := 0; ; i++ {
		// handle separator
		if i > 0 {
			gp.parseOptionalSpacesOrComments(ps)
			pos3 := ps.KeepPos()
			if err := ps.M.Rune('|'); err != nil {
				pos3.Restore()
				if i == 1 {
					return nil // ok, just not an OR
				}

				res := &OrRule{}
				res.childs_ = w
				res.SetPos(pos0.Pos, ps.Pos)
				ps.Node = res

				return nil // ok
			}

			gp.parseOptionalSpacesOrComments(ps)
		}

		// precedence tree construction ("and" is higher vs "or")
		if err := gp.parseAndRule(ps); err != nil {
			if i == 0 {
				return err // fail, no rule
			}
			return err // fail, not expecting error after sep
		}

		resRule := ps.Node.(Rule)
		w = append(w, resRule)
	}
}
func (gp *grammarParser) parseAndRule(ps *PState) error {
	pos0 := ps.KeepPos()
	w := []Rule{}
	for i := 0; ; i++ {
		// handle separator
		pos2 := ps.KeepPos()
		if i > 0 {
			gp.parseOptionalSpacesOrComments(ps)
		}

		if err := gp.parseBasicItemRule(ps); err != nil {
			if i == 0 {
				return err // fail, no rule
			}
			if i == 1 {
				return nil // ok, just not an AND
			}
			pos2.Restore()
			break // ok, don't include the spaces
		}

		resRule := ps.Node.(Rule)
		w = append(w, resRule)
	}

	res := &AndRule{}
	res.childs_ = w
	res.SetPos(pos0.Pos, ps.Pos)
	ps.Node = res

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
	callRuleSym := "@"
	if err := ps.M.Sequence(callRuleSym); err != nil {
		return err, false
	}

	// name
	name, err := gp.parseName(ps)
	if err != nil {
		return err, true
	}

	// args
	if err := ps.M.Rune('('); err != nil {
		return err, true
	}
	parseProcRuleArg := func() (ProcRuleArg, error) {
		err := gp.parseItemRule(ps)
		if err != nil {
			// special case: try to parse an int as a direct arg
			if v, err, ok := gp.parseInt(ps); ok && err == nil {
				return v, nil
			}
		}
		return ps.Node, err
	}
	args := []ProcRuleArg{}
	for i := 0; ; i++ {
		arg, err := parseProcRuleArg()
		if err != nil {
			if i == 0 {
				break
			}
			return err, true
		}
		args = append(args, arg)
		gp.parseOptionalSpacesOrComments(ps)
		if err := ps.M.Rune(','); err != nil {
			break
		}
	}
	if err := ps.M.Rune(')'); err != nil {
		return err, true
	}

	res := &ProcRule{}
	res.name = name
	res.args = args
	res.SetPos(i0, ps.Pos)
	ps.Node = res

	return nil, true
}
func (gp *grammarParser) parseRefRule(ps *PState) (error, bool) {
	i0 := ps.Pos
	name, err := gp.parseName(ps)
	if err != nil {
		return nil, false // err is lost
	}
	res := &RefRule{name: name}
	res.SetPos(i0, ps.Pos)
	ps.Node = res
	return nil, true
}

func (gp *grammarParser) parseStringRule(ps *PState) (error, bool) {
	pos0 := ps.KeepPos()

	esc := '\\' // alows to escape the quote
	if err := ps.M.StringSection("\"", esc, true, 1000, false); err != nil {
		return nil, false
	}

	// needed: ex: transforms "\n" (2 runes) into a single '\n'
	str := string(pos0.Bytes())
	u, err := strconv.Unquote(str)
	if err != nil {
		return err, true
	}

	sr := &StringRule{}
	sr.runes = []rune(u)
	sr.SetPos(pos0.Pos, ps.Pos)
	ps.Node = sr
	return nil, true
}
func (gp *grammarParser) parseParenRule(ps *PState) (error, bool) {
	pos0 := ps.KeepPos()
	if err := ps.M.Rune('('); err != nil {
		return err, false
	}
	gp.parseOptionalSpacesOrComments(ps)
	if err := gp.parseItemRule(ps); err != nil {
		return err, true
	}
	gp.parseOptionalSpacesOrComments(ps)
	ruleX := ps.Node.(Rule)
	if err := ps.M.Rune(')'); err != nil {
		return err, true
	}

	// option rune
	pt := parenRTNone
	pos2 := ps.KeepPos()
	ru, err := ps.ReadRune()
	if err == nil {
		u := parenRType(ru)
		switch u {
		case parenRTNone,
			parenRTOptional,
			parenRTZeroOrMore,
			parenRTOneOrMore,
			parenRTStrMid,
			parenRTStrOr,
			parenRTStrOrRange,
			parenRTStrOrNeg:
			pt = u
		default:
			pos2.Restore()
		}
	}

	u := &ParenRule{typ: pt}
	u.setOnlyChild(ruleX)
	u.SetPos(pos0.Pos, ps.Pos)
	ps.Node = u

	return nil, true
}

//----------

func (gp *grammarParser) parseInt(ps *PState) (int, error, bool) {
	pos0 := ps.KeepPos()
	if err := ps.M.Integer(); err != nil {
		return 0, err, false
	}

	u := string(pos0.Bytes())
	v, err := strconv.ParseInt(u, 10, 64)
	if err != nil {
		return 0, err, true
	}

	n := &BasicPNode{}
	n.SetPos(pos0.Pos, ps.Pos)
	ps.Node = n
	return int(v), nil, true
}

//----------

func (gp *grammarParser) parseOptionalSpacesOrComments(ps *PState) {
	for {
		if ps.M.SpacesIncludingNL() {
			continue
		}
		if gp.parseComments(ps) {
			continue
		}
		break
	}
}
func (gp *grammarParser) parseComments(ps *PState) bool {
	if err := ps.M.Rune('#'); err == nil {
		_ = ps.M.ToNLIncludeOrEnd(0)
		return true
	}
	if err := ps.M.Sequence("//"); err == nil {
		_ = ps.M.ToNLIncludeOrEnd(0)
		return true
	}
	return false
}

//func (gp *grammarParser) parseEmptyLine(ps *PState) bool {
//	if err := ps.MatchRune('\n'); err == nil {
//		return true
//	}
//	return false
//}
