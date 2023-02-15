package toolbarparser

import (
	"unicode"

	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil/pscan"
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

type varDeclParser struct {
	sc *pscan.Scanner
}

func newVarDeclParser() *varDeclParser {
	p := &varDeclParser{}
	p.sc = pscan.NewScanner()
	return p
}
func (p *varDeclParser) parseVarDecl(src []byte) (*VarDecl, error) {
	p.sc.SetSrc(src)
	if v, p2, err := p.sc.M.OrValue(0,
		p.parseTildeVarDecl,
		p.parseDollarVarDecl,
	); err != nil {
		return nil, p.sc.SrcError(p2, err)
	} else {
		return v.(*VarDecl), nil
	}
}
func (p *varDeclParser) parseTildeVarDecl(pos int) (any, int, error) {
	nameRe := "~(0|[1-9][0-9]*)"
	vd := &VarDecl{}
	vk := p.sc.NewValueKeepers(2)
	if p2, err := p.sc.M.And(pos,
		// name
		vk[0].WKeepValue(p.sc.W.StringValue(p.sc.W.RegexpFromStartCached(nameRe, 100))),
		// value
		p.sc.W.Rune('='),
		vk[1].WKeepValue(p.parseVarValue),
	); err != nil {
		return nil, p2, err
	} else {
		vd.Name = vk[0].V.(string)
		vd.Value = vk[1].V.(string)
		return vd, p2, err
	}
}
func (p *varDeclParser) parseDollarVarDecl(pos int) (any, int, error) {
	nameRe := "\\$[_a-zA-Z0-9]+"
	vk := p.sc.NewValueKeepers(2)
	if p2, err := p.sc.M.And(pos,
		// name
		vk[0].WKeepValue(p.sc.W.StringValue(p.sc.W.RegexpFromStartCached(nameRe, 100))),
		// value (optional after =)
		p.sc.W.Rune('='),
		p.sc.W.Optional(vk[1].WKeepValue(p.parseVarValue)),
	); err != nil {
		return nil, p2, err
	} else {
		vd := &VarDecl{}
		vd.Name = vk[0].V.(string)
		if vk[1].V != nil {
			vd.Value = vk[1].V.(string)
		}
		return vd, p2, err
	}
}

//----------

func (p *varDeclParser) parseVarValue(pos int) (any, int, error) {
	notSpace := func(ru rune) bool { return !unicode.IsSpace(ru) }
	if v, p2, err := p.sc.M.StringValue(pos, p.sc.W.Loop(p.sc.W.Or(
		p.sc.W.EscapeAny(osutil.EscapeRune),
		p.sc.W.QuotedString2('\\', 3000, 3000),
		p.sc.W.RuneFn(notSpace),
	))); err != nil {
		return nil, p2, err
	} else {
		return v.(string), p2, nil
	}
}
