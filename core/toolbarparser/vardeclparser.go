package toolbarparser

import (
	"unicode"

	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil/btparser"
)

func parseVarDecl(src string) (*VarDecl, error) {
	p := getVarDeclParser()
	return p.parseVarDecl([]byte(src))
}

//----------

var vdp *varDeclParser

func getVarDeclParser() *varDeclParser {
	if vdp == nil {
		vdp = newVarDeclParser()
	}
	return vdp
}

//----------
//----------
//----------

const varDeclDataKey = "toolbarparser.vardecl"

//----------

type varDeclParser struct {
	g  btparser.Rules
	fn btparser.MFn
}

func newVarDeclParser() *varDeclParser {
	p := &varDeclParser{}
	p.g = btparser.NewRules()
	p.build()
	return p
}

func (p *varDeclParser) parseVarDecl(src []byte) (*VarDecl, error) {
	vd := &VarDecl{}

	ps := btparser.NewParserStateFromBytes(src)
	ps.UserData[varDeclDataKey] = vd

	if _, err := p.g.Parse(ps, p.fn); err != nil {
		return nil, err
	}
	return vd, nil
}

func (p *varDeclParser) build() {
	g := p.g

	varDeclData := func(ps *btparser.ParserState) *VarDecl {
		vd, ok := ps.UserData[varDeclDataKey].(*VarDecl)
		if !ok {
			panic("vardecl parser missing VarDecl userdata")
		}
		return vd
	}
	assignName := func(fn btparser.MFn) btparser.MFn {
		return btparser.AssignFn(
			func(ps *btparser.ParserState) *string {
				return &varDeclData(ps).Name
			},
			g.VString(fn),
		)
	}
	assignValue := func(fn btparser.MFn) btparser.MFn {
		return btparser.AssignFn(
			func(ps *btparser.ParserState) *string {
				return &varDeclData(ps).Value
			},
			g.VString(fn),
		)
	}

	//----------

	tildeDigits := g.Or(
		g.Rune('0'),
		g.And(
			g.DigitNotZero(),
			g.Optional(g.Digits()),
		),
	)
	tildeName := g.And(
		g.Rune('~'),
		tildeDigits,
	)
	dollarName := g.And(
		g.Rune('$'),
		g.Loop1(g.Or(
			g.Rune('_'),
			g.AsciiLetter(),
			g.Digit(),
		)),
	)
	varValue := g.Loop1(g.Or(
		g.Escape(osutil.EscapeRune),
		g.QuotedString2('\\', 3000, 3000),
		g.RuneFn(func(ru rune) bool {
			return !unicode.IsSpace(ru)
		}),
	))
	tildeDecl := g.And(
		assignName(tildeName),
		g.Rune('='),
		assignValue(varValue),
	)
	dollarDecl := g.And(
		assignName(dollarName),
		g.Rune('='),
		g.Optional(assignValue(varValue)),
	)

	p.fn = g.And(
		g.Or(
			tildeDecl,
			dollarDecl,
		),
		g.Eof(),
	)
}
