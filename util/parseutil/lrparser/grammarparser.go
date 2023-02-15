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
		gp.sc.W.Loop(gp.sc.W.Or(
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
	vkName := gp.sc.NewValueKeeper()
	vkItem := gp.sc.NewValueKeeper()
	p2, err := gp.sc.M.And(pos,
		gp.sc.W.Optional(gp.sc.W.And(
			gp.sc.W.Sequence(defRuleStartSym),
			func(pos int) (int, error) { isStart = true; return pos, nil },
		)),
		gp.sc.W.Optional(gp.sc.W.And(
			gp.sc.W.Sequence(defRuleNoPrintSym),
			func(pos int) (int, error) { isNoPrint = true; return pos, nil },
		)),
		vkName.WKeepValue(gp.parseName),
		gp.parseOptSpacesOrComments,
		gp.sc.W.FatalOnError("expecting '='",
			gp.sc.W.Rune('='),
		),
		gp.parseOptSpacesOrComments,
		vkItem.WKeepValue(gp.parseItemRule),
		gp.parseOptSpacesOrComments,
		gp.sc.W.FatalOnError("expecting close rule \";\"?",
			gp.sc.W.Rune(';'),
		),
	)
	if err != nil {
		return p2, err
	}

	name := vkName.V.(string)
	if gp.ri.has(name) {
		return p2, fmt.Errorf("rule already defined: %v", name)
	}

	// setup
	dr := &DefRule{name: name, isStart: isStart, isNoPrint: isNoPrint}
	dr.setOnlyChild(vkItem.V.(Rule))
	dr.SetPos(pos0, p2)
	gp.ri.set(dr.name, dr)

	return p2, nil
}
func (gp *grammarParser) parseName(pos int) (any, int, error) {
	u := "[_a-zA-Z][_a-zA-Z0-9$]*"
	return gp.sc.M.StringValue(pos, gp.sc.W.RegexpFromStartCached(u, 100))
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

	vk := gp.sc.NewValueKeepers(3)
	p2, err := gp.sc.M.And(pos,
		gp.sc.W.Sequence("if"),
		gp.parseSpacesOrComments,
		gp.sc.W.FatalOnError("expecting name",
			vk[0].WKeepValue(gp.parseRefRule),
		),
		// then
		gp.parseOptSpacesOrComments,
		gp.sc.W.FatalOnError("expecting '?'",
			gp.sc.W.Rune('?'),
		),
		gp.parseOptSpacesOrComments,
		vk[1].WKeepValue(gp.parseItemRule),
		// else
		gp.parseOptSpacesOrComments,
		gp.sc.W.FatalOnError("expecting ':'",
			gp.sc.W.Rune(':'),
		),
		gp.parseOptSpacesOrComments,
		vk[2].WKeepValue(gp.parseItemRule),
	)
	if err != nil {
		return nil, p2, err
	}

	// setup
	res := &IfRule{}
	res.addChilds(vk[0].V.(Rule))
	res.addChilds(vk[1].V.(Rule))
	res.addChilds(vk[2].V.(Rule))
	res.SetPos(pos0, p2)
	return res, p2, nil
}

//----------

func (gp *grammarParser) parseOrTreeRule(pos int) (any, int, error) {
	w := []Rule{}
	if p2, err := gp.sc.M.LoopSep(pos,
		gp.sc.W.OnValue(
			// precedence tree construction ("and" is higher precedence than "or")
			gp.parseAndTreeRule,
			func(v any) { w = append(w, v.(Rule)) },
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
	if p2, err := gp.sc.M.Loop(pos, gp.sc.W.And(
		gp.parseOptSpacesOrComments,
		gp.sc.W.OnValue(
			gp.parseBasicItemRule,
			func(v any) { w = append(w, v.(Rule)) },
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
	vkName := gp.sc.NewValueKeeper()
	args := []ProcRuleArg{}
	if p2, err := gp.sc.M.And(pos,
		gp.sc.W.Sequence("@"),
		vkName.WKeepValue(gp.parseName),
		gp.sc.W.Rune('('),
		gp.parseOptSpacesOrComments,
		gp.sc.W.LoopSep(
			gp.sc.W.OnValue(
				gp.sc.W.OrValue(
					gp.parseItemRule,
					gp.sc.M.IntValue,
				),
				func(v any) { args = append(args, v.(ProcRuleArg)) },
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
		res.name = vkName.V.(string)
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
	if v, p2, err := gp.sc.M.StringValue(pos,
		gp.sc.W.StringSection("\"", '\\', true, 1000, false),
	); err != nil {
		return nil, p2, err
	} else {
		// needed: ex: transforms "\n" (2 runes) into a single '\n'
		str := v.(string)
		u, err := strconv.Unquote(str)
		if err != nil {
			return nil, pos, gp.sc.EnsureFatalError(err)
		}

		sr := &StringRule{}
		sr.runes = []rune(u)
		sr.SetPos(pos, p2)
		return sr, p2, nil
	}
}
func (gp *grammarParser) parseParenRule(pos int) (any, int, error) {
	vk := gp.sc.NewValueKeepers(2)
	if p2, err := gp.sc.M.And(pos,
		gp.sc.W.Rune('('),
		gp.parseOptSpacesOrComments,
		vk[0].WKeepValue(gp.parseItemRule),
		gp.parseOptSpacesOrComments,
		gp.sc.W.Rune(')'),
		gp.sc.W.Optional(vk[1].WKeepValue(gp.sc.W.RuneValue(gp.sc.W.Or(
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
		res.setOnlyChild(vk[0].V.(Rule))
		if vk[1].V != nil {
			res.typ = parenRType(vk[1].V.(rune))
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
	return gp.sc.M.Loop(pos, gp.sc.W.Or(
		gp.sc.W.Spaces(true, 0),
		gp.parseComments,
	))
}
func (gp *grammarParser) parseComments(pos int) (int, error) {
	return gp.sc.M.And(pos,
		gp.sc.W.Or(
			gp.sc.W.Rune('#'),
			gp.sc.W.Sequence("//"),
		),
		gp.sc.W.ToNLOrErr(true, 0),
	)
}
