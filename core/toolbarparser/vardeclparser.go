package toolbarparser

import (
	"unicode"

	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil"
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
	sc *parseutil.Scanner
}

func newVarDeclParser() *varDeclParser {
	p := &varDeclParser{}
	p.sc = parseutil.NewScanner()
	return p
}
func (p *varDeclParser) parseVarDecl(src []byte) (*VarDecl, error) {
	p.sc.SetSrc(src)
	vk := p.sc.NewValueKeeper()
	if err := p.sc.P.Or(
		vk.KeepValue(p.parseTildeVarDecl),
		vk.KeepValue(p.parseDollarVarDecl),
	)(); err != nil {
		//return nil, p.sc.SrcError(err)
		return nil, err
	}
	return vk.Value.(*VarDecl), nil
}
func (p *varDeclParser) parseTildeVarDecl() (any, error) {
	nameRe := "~(0|[1-9][0-9]*)"
	vd := &VarDecl{}
	pos0 := p.sc.KeepPos()
	vk := p.sc.NewValueKeeper()
	err := p.sc.P.And(
		// name
		p.sc.P.RegexpFromStartCached(nameRe, 100),
		func() error {
			vd.Name = string(pos0.Bytes())
			return nil
		},
		// value
		p.sc.P.Rune('='),
		vk.KeepValue(p.parseVarValue),
	)()
	vd.Value = vk.StringOptional()
	return vd, err
}
func (p *varDeclParser) parseDollarVarDecl() (any, error) {
	nameRe := "\\$[_a-zA-Z0-9]+"
	vd := &VarDecl{}
	pos0 := p.sc.KeepPos()
	vk := p.sc.NewValueKeeper()
	err := p.sc.P.And(
		// name
		p.sc.P.RegexpFromStartCached(nameRe, 100),
		func() error {
			vd.Name = string(pos0.Bytes())
			return nil
		},
		// value (optional after =)
		p.sc.P.Rune('='),
		p.sc.P.Optional(vk.KeepValue(p.parseVarValue)),
	)()
	vd.Value = vk.StringOptional()
	return vd, err
}

//----------

func (p *varDeclParser) parseVarValue() (any, error) {
	pos0 := p.sc.KeepPos()
	cf := p.sc.P.GetCacheFunc("parseVarValue") // minor performance improvement; also here for example of usage
	if !cf.IsSet() {
		notSpace := func(ru rune) bool { return !unicode.IsSpace(ru) }
		cf.Set(p.sc.P.Loop(
			p.sc.P.Or(
				p.sc.P.EscapeAny(osutil.EscapeRune),
				p.sc.P.QuotedString2('\\', 3000, 3000),
				p.sc.P.RuneFn(notSpace),
			),
			nil, false,
		))
	}
	if err := cf.Run(); err != nil {
		return "", err
	}
	return string(pos0.Bytes()), nil
}
