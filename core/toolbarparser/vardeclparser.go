package toolbarparser

import (
	"fmt"
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
	ru, err := p.sc.PeekRune()
	if err != nil {
		return nil, err
	}
	switch ru {
	case '~':
		return p.parseTildeVarDecl()
	case '$':
		return p.parseDollarVarDecl()
	}
	return nil, fmt.Errorf("unexpected rune: %v", ru)
}
func (p *varDeclParser) parseTildeVarDecl() (*VarDecl, error) {
	pos0 := p.sc.KeepPos()
	// name
	u := "~(0|[1-9][0-9]*)"
	if err := p.sc.M.RegexpFromStartCached(u); err != nil {
		return nil, err
	}
	name := string(p.sc.BytesFrom(pos0.Pos))

	w := &VarDecl{Name: name}

	// assign
	if err := p.sc.M.Rune('='); err != nil {
		return nil, fmt.Errorf("expecting assign") //err
	}
	// value
	v, err := p.parseVarValue()
	if err != nil {
		return nil, err
	}
	w.Value = v
	return w, nil
}
func (p *varDeclParser) parseDollarVarDecl() (*VarDecl, error) {
	pos0 := p.sc.KeepPos()
	// name
	u := "\\$[_a-zA-Z0-9]+"
	if err := p.sc.M.RegexpFromStartCached(u); err != nil {
		return nil, err
	}
	name := string(p.sc.BytesFrom(pos0.Pos))

	w := &VarDecl{Name: name}

	// assign (optional)
	pos2 := p.sc.KeepPos()
	if err := p.sc.M.Rune('='); err != nil {
		if p.sc.M.Eof() {
			pos2.Restore()
			return w, nil
		}
		return nil, err
	}
	// value
	v, err := p.parseVarValue()
	if err != nil {
		return nil, err
	}
	w.Value = v
	return w, nil
}

//----------

func (p *varDeclParser) parseVarValue() (string, error) {
	pos0 := p.sc.KeepPos()
	err := p.sc.RestorePosOnErr(func() error {
		// any runes (with some exceptions)
		notSpace := func(ru rune) bool { return !unicode.IsSpace(ru) }
		for {
			// TODO: necessary?
			//if err := p.sc.M.Eof(); err == nil {
			//	break
			//}

			if err := p.sc.M.EscapeAny(osutil.EscapeRune); err == nil {
				continue
			}
			if err := p.sc.M.QuotedString2('\\', 3000, 3000); err == nil {
				continue
			}
			if err := p.sc.M.RuneFn(notSpace); err == nil {
				continue
			}
			break
		}
		if pos0.IsEmpty() {
			return fmt.Errorf("empty")
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return string(p.sc.BytesFrom(pos0.Pos)), nil
}
