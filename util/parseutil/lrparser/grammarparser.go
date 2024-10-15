package lrparser

import (
	"fmt"
	"strconv"

	"github.com/jmigpin/editor/util/parseutil/pscan"
)

type grammarParser struct {
	ri *RuleIndex
	sc *pscan.Scanner
}

func newGrammarParser(ri *RuleIndex) *grammarParser {
	gp := &grammarParser{ri: ri}
	gp.sc = pscan.NewScanner()
	return gp
}
func (gp *grammarParser) parse(fset *FileSet) error {
	gp.sc.SetSrc(fset.Src)

	// rules are parsed into the ruleindex (parse result)

	if p2, err := gp.parseRules(0); err != nil {
		return gp.sc.SrcError(p2, err)
	}
	return nil
}
func (gp *grammarParser) parseRules(pos int) (int, error) {
	return gp.sc.M.And(pos,
		gp.sc.W.LoopOneOrMore(gp.sc.W.Or(
			gp.parseSpacesOrComments,
			gp.parseDefRule,
		)),
		gp.sc.W.Eof(),
	)
}
func (gp *grammarParser) parseDefRule(pos int) (int, error) {
	pos0 := pos

	isStart := false
	isNoPrint := false
	name := ""
	item := (Rule)(nil)
	p2, err := gp.sc.M.And(pos,
		gp.sc.W.Optional(gp.sc.W.And(
			gp.sc.W.Sequence(defRuleStartSym),
			func(pos int) (int, error) { isStart = true; return pos, nil },
		)),
		gp.sc.W.Optional(gp.sc.W.And(
			gp.sc.W.Sequence(defRuleNoPrintSym),
			func(pos int) (int, error) { isNoPrint = true; return pos, nil },
		)),
		pscan.WKeep(&name, gp.parseName),
		gp.parseOptSpacesOrComments,
		gp.sc.W.FatalOnError("expecting '='",
			gp.sc.W.Rune('='),
		),
		gp.parseOptSpacesOrComments,
		pscan.WKeep(&item, gp.parseItemRule),
		gp.parseOptSpacesOrComments,
		gp.sc.W.FatalOnError("expecting close rule \";\"?",
			gp.sc.W.Rune(';'),
		),
	)
	if err != nil {
		return p2, err
	}

	if gp.ri.has(name) {
		return p2, fmt.Errorf("rule already defined: %v", name)
	}

	// setup
	dr := &DefRule{name: name, isStart: isStart, isNoPrint: isNoPrint}
	dr.setOnlyChild(item)
	dr.SetPos(pos0, p2)
	gp.ri.set(dr.name, dr)

	return p2, nil
}
func (gp *grammarParser) parseName(pos int) (any, int, error) {
	u := "[_a-zA-Z][_a-zA-Z0-9$]*"
	return gp.sc.M.StrValue(pos, gp.sc.W.RegexpFromStartCached(u, 100))
}

//----------

func (gp *grammarParser) parseItemRule(pos int) (any, int, error) {
	return gp.sc.M.OrValue(pos,
		gp.parseIfRule,
		gp.parseOrTreeRule, // precedence tree
	)
}

//----------

func (gp *grammarParser) parseIfRule(pos int) (any, int, error) {
	pos0 := pos

	rules := [3]Rule{}
	p2, err := gp.sc.M.And(pos,
		gp.sc.W.Sequence("if"),
		gp.parseSpacesOrComments,
		gp.sc.W.FatalOnError("expecting name",
			pscan.WKeep(&rules[0], gp.parseRefRule),
		),
		// then
		gp.parseOptSpacesOrComments,
		gp.sc.W.FatalOnError("expecting '?'",
			gp.sc.W.Rune('?'),
		),
		gp.parseOptSpacesOrComments,
		pscan.WKeep(&rules[1], gp.parseItemRule),
		// else
		gp.parseOptSpacesOrComments,
		gp.sc.W.FatalOnError("expecting ':'",
			gp.sc.W.Rune(':'),
		),
		gp.parseOptSpacesOrComments,
		pscan.WKeep(&rules[2], gp.parseItemRule),
	)
	if err != nil {
		return nil, p2, err
	}

	// setup
	res := &IfRule{}
	res.addChilds(rules[:]...)
	res.SetPos(pos0, p2)
	return res, p2, nil
}

//----------

func (gp *grammarParser) parseOrTreeRule(pos int) (any, int, error) {
	w := []Rule{}
	if p2, err := gp.sc.M.LoopSep(pos, false,
		pscan.WOnValueM(
			// precedence tree construction ("and" is higher precedence than "or")
			gp.parseAndTreeRule,
			func(v Rule) error { w = append(w, v); return nil },
		),
		// separator
		gp.sc.W.And(
			gp.parseOptSpacesOrComments,
			gp.sc.W.Rune('|'),
			gp.parseOptSpacesOrComments,
		),
	); err != nil {
		return nil, p2, err
	} else {
		if len(w) == 1 {
			return w[0], p2, nil // just the "and" rule
		}
		res := &OrRule{}
		res.childs2 = w
		res.SetPos(pos, p2)
		return res, p2, nil
	}
}
func (gp *grammarParser) parseAndTreeRule(pos int) (any, int, error) {
	w := []Rule{}
	// NOTE: better than using a loopsep because it doesn't include the end spaces
	if p2, err := gp.sc.M.LoopOneOrMore(pos, gp.sc.W.And(
		gp.parseOptSpacesOrComments,
		pscan.WOnValueM(
			gp.parseBasicItemRule,
			func(v Rule) error { w = append(w, v); return nil },
		),
	)); err != nil {
		return nil, p2, err
	} else {
		if len(w) == 1 {
			return w[0], p2, nil // just the "basic" rule
		}
		res := &AndRule{}
		res.childs2 = w
		res.SetPos(pos, p2)
		return res, p2, nil
	}
}

//----------

func (gp *grammarParser) parseBasicItemRule(pos int) (any, int, error) {
	return gp.sc.M.OrValue(pos,
		gp.parseProcRule,
		gp.parseRefRule,
		gp.parseStringRule,
		gp.parseParenRule,
	)
}
func (gp *grammarParser) parseProcRule(pos int) (any, int, error) {
	name := ""
	args := []ProcRuleArg{}
	if p2, err := gp.sc.M.And(pos,
		gp.sc.W.Sequence("@"),
		pscan.WKeep(&name, gp.parseName),
		gp.sc.W.Rune('('),
		gp.parseOptSpacesOrComments,
		gp.sc.W.LoopSep(
			false,
			pscan.WOnValueM(
				gp.sc.W.OrValue(
					gp.parseItemRule,
					gp.sc.M.IntValue,
				),
				func(v ProcRuleArg) error { args = append(args, v); return nil },
			),
			// separator
			gp.sc.W.And(
				gp.parseOptSpacesOrComments,
				gp.sc.W.Rune(','),
				gp.parseOptSpacesOrComments,
			),
		),
		gp.parseOptSpacesOrComments,
		gp.sc.W.Rune(')'),
	); err != nil {
		return nil, p2, err
	} else {
		res := &ProcRule{}
		res.name = name
		res.args = args
		res.SetPos(pos, p2)
		return res, p2, nil
	}
}
func (gp *grammarParser) parseRefRule(pos int) (any, int, error) {
	if v, p2, err := gp.parseName(pos); err != nil {
		return nil, p2, err
	} else {
		res := &RefRule{name: v.(string)}
		res.SetPos(pos, p2)
		return res, p2, nil
	}
}

func (gp *grammarParser) parseStringRule(pos int) (any, int, error) {
	if v, p2, err := gp.sc.M.StrValue(pos,
		gp.sc.W.StringSection("\"", '\\', true, 1000, false),
	); err != nil {
		return nil, p2, err
	} else {
		// needed: ex: transforms "\n" (2 runes) into a single '\n'
		str := v.(string)
		u, err := strconv.Unquote(str)
		if err != nil {
			return nil, pos, pscan.FatalError(err)
		}

		sr := &StringRule{}
		sr.runes = []rune(u)
		sr.SetPos(pos, p2)
		return sr, p2, nil
	}
}
func (gp *grammarParser) parseParenRule(pos int) (any, int, error) {
	rule := (Rule)(nil)
	ru := (*rune)(nil)
	if p2, err := gp.sc.M.And(pos,
		gp.sc.W.Rune('('),
		gp.parseOptSpacesOrComments,
		pscan.WKeep(&rule, gp.parseItemRule),
		gp.parseOptSpacesOrComments,
		gp.sc.W.Rune(')'),
		gp.sc.W.Optional(pscan.WKeep(&ru, gp.sc.W.RuneValue(gp.sc.W.Or(
			gp.sc.W.Rune(rune(parenRTOptional)),
			gp.sc.W.Rune(rune(parenRTZeroOrMore)),
			gp.sc.W.Rune(rune(parenRTOneOrMore)),
			gp.sc.W.Rune(rune(parenRTStrMid)),
			gp.sc.W.Rune(rune(parenRTStrOr)),
			gp.sc.W.Rune(rune(parenRTStrOrRange)),
			gp.sc.W.Rune(rune(parenRTStrOrNeg)),
		)))),
	); err != nil {
		return nil, p2, err
	} else {
		res := &ParenRule{}
		res.setOnlyChild(rule)
		if ru != nil {
			res.typ = parenRType(*ru)
		}
		res.SetPos(pos, p2)
		return res, p2, nil
	}
}

//----------

func (gp *grammarParser) parseOptSpacesOrComments(pos int) (int, error) {
	return gp.sc.M.Optional(pos, gp.parseSpacesOrComments)
}
func (gp *grammarParser) parseSpacesOrComments(pos int) (int, error) {
	return gp.sc.M.LoopOneOrMore(pos, gp.sc.W.Or(
		gp.sc.M.SpacesIncludingNewline,
		gp.parseComments,
	))
}
func (gp *grammarParser) parseComments(pos int) (int, error) {
	return gp.sc.M.And(pos,
		gp.sc.W.Or(
			gp.sc.W.Rune('#'),
			gp.sc.W.Sequence("//"),
		),
		gp.sc.W.LoopUntilNLOrEof(-1, true, 0),
	)
}
