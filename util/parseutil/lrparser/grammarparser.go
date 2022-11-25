package lrparser

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/jmigpin/editor/util/parseutil"
)

type grammarParser struct {
	ri *RuleIndex
}

func newGrammarParser(ri *RuleIndex) *grammarParser {
	gp := &grammarParser{ri: ri}
	return gp
}
func (gp *grammarParser) parse(fset *FileSet) error {
	ps := parseutil.NewPState(fset.Src)
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
	if err := ps.MatchEof(); err == nil {
		return false, nil
	}
	if err := gp.parseDefRule(ps); err != nil {
		return false, err
	}
	return true, nil
}
func (gp *grammarParser) parseDefRule(ps *PState) error {
	i0 := ps.Pos

	// options
	isStart := false
	if err := ps.MatchString(defRuleStartSym); err == nil {
		isStart = true
	}
	isNoPrint := false
	if err := ps.MatchString(defRuleNoPrintSym); err == nil {
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

	if err := ps.MatchRune('='); err != nil {
		return errors.New("expecting =")
	}

	gp.parseOptionalSpacesOrComments(ps)

	if err := gp.parseItemRule(ps); err != nil {
		return err
	}

	gp.parseOptionalSpacesOrComments(ps)

	if err := ps.MatchRune(';'); err != nil {
		return errors.New("expecting close rule \";\"?")
	}

	// setup
	dr := &DefRule{name: name, isStart: isStart, isNoPrint: isNoPrint}
	dr.setOnlyChild(ps.Node.(Rule))
	dr.SetPos(i0, ps.Pos)
	gp.ri.set(dr.name, dr)

	return nil
}
func (gp *grammarParser) parseName(ps *PState) (string, error) {
	name := ""
	ps2 := ps.Copy()
	for i := 0; ; i++ {
		ps3 := ps2.Copy()
		ru, err := ps3.ReadRune()
		if err != nil {
			break
		}
		valid := unicode.IsLetter(ru) || strings.Contains("_$", string(ru))
		if !valid && i > 0 {
			valid = unicode.IsDigit(ru)
		}
		if !valid {
			break
		}
		name += string(ru)
		ps2.Set(ps3)
	}
	if name == "" {
		return "", fmt.Errorf("empty name")
	}
	ps.Set(ps2)
	return name, nil
}

//----------

func (gp *grammarParser) parseItemRule(ps *PState) error {
	// taking into consideration precedence tree construction

	if err, ok := gp.parseIfRule(ps); ok {
		return err
	}
	return gp.parseOrRule(ps)
}

//----------

func (gp *grammarParser) parseIfRule(ps *PState) (error, bool) {
	if err := ps.MatchString("if "); err != nil {
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
	res.addChilds(nameRef)
	res.addChilds(thenRule)
	res.addChilds(elseRule)
	res.SetPos(i0, ps.Pos)
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
				res.SetPos(i0, ps2.Pos)
				ps2.Node = res

				ps.Set(ps2)
				return nil // ok
			}

			gp.parseOptionalSpacesOrComments(ps2)
		}

		// precedence tree construction ("and" is higher vs "or")
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
	pos0 := ps.Pos
	ps2 := ps.Copy()
	w := []Rule{}
	for i := 0; ; i++ {
		// handle separator
		if i > 0 {
			gp.parseOptionalSpacesOrComments(ps2)
		}

		if err := gp.parseBasicItemRule(ps2); err != nil {
			if i == 0 {
				//ps.Set(ps2) // better error?
				return err // fail, no rule
			}
			if i == 1 {
				ps.Set(ps2)
				return nil // ok, just not an AND
			}
			break // ok, don't include the spaces
		}

		resRule := ps2.Node.(Rule)
		w = append(w, resRule)
		ps.Set(ps2) // will show errors more advanced
	}
	//ps.Set(ps2)

	res := &AndRule{}
	res.childs_ = w
	res.SetPos(pos0, ps.Pos)
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
	if err := ps.MatchString(callRuleSym); err != nil {
		return err, false
	}

	// name
	name, err := gp.parseName(ps)
	if err != nil {
		return err, true
	}

	// args
	if err := ps.MatchRune('('); err != nil {
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
		if err := ps.MatchRune(','); err != nil {
			break
		}
	}
	if err := ps.MatchRune(')'); err != nil {
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
	i0 := ps.Pos

	q := "\""
	esc := '\\' // alows to escape the quote
	if err := ps.StringSection(q, q, esc, true, 1000, false); err != nil {
		return nil, false
	}
	sr := &StringRule{}
	sr.SetPos(i0, ps.Pos)

	// needed: ex: transforms "\n" (2 runes) into a single '\n'
	s := sr.SrcString(ps.Src)
	u, err := strconv.Unquote(s)
	if err != nil {
		return err, true
	}
	sr.runes = []rune(u)

	ps.Node = sr
	return nil, true
}
func (gp *grammarParser) parseParenRule(ps *PState) (error, bool) {
	i0 := ps.Pos
	if err := ps.MatchRune('('); err != nil {
		return err, false
	}
	gp.parseOptionalSpacesOrComments(ps)
	if err := gp.parseItemRule(ps); err != nil {
		return err, true
	}
	gp.parseOptionalSpacesOrComments(ps)
	ruleX := ps.Node.(Rule)
	if err := ps.MatchRune(')'); err != nil {
		return err, true
	}

	// option rune
	pt := parenRTNone
	ps2 := ps.Copy()
	ru, err := ps2.ReadRune()
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
			ps.Set(ps2) // advance
		}
	}

	u := &ParenRule{typ: pt}
	u.setOnlyChild(ruleX)
	u.SetPos(i0, ps.Pos)
	ps.Node = u

	return nil, true
}

//----------

func (gp *grammarParser) parseInt(ps *PState) (int, error, bool) {
	pos0 := ps.Pos
	rrs09 := RuneRanges{RuneRange{'0', '9'}}
	ps2 := ps.Copy()
	for i := 0; ; i++ {
		if i == 0 {
			if err := ps2.MatchRunesOr([]rune("-+")); err == nil {
				continue
			}
		}
		if err := ps2.MatchRuneRanges(rrs09); err == nil {
			continue
		}
		break
	}

	n := &BasicPNode{}
	n.SetPos(pos0, ps2.Pos)
	if n.PosEmpty() {
		return 0, nil, false
	}

	u := n.SrcString(ps2.Src)
	v, err := strconv.ParseInt(u, 10, 64)
	if err != nil {
		return 0, err, true
	}

	ps2.Node = n
	ps.Set(ps2)
	return int(v), nil, true
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
