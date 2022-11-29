package toolbarparser

import (
	"fmt"

	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil"
)

func parseVarRefs(src []byte) ([]*VarRef, error) {
	p := getVarRefParser()
	return p.parseVarRefs(src)
}

//----------

var vrp *varRefParser

func getVarRefParser() *varRefParser {
	if vrp == nil {
		vrp = newVarRefParser()
	}
	return vrp
}

//----------
//----------
//----------

type varRefParser struct {
	sc *parseutil.Scanner
}

func newVarRefParser() *varRefParser {
	p := &varRefParser{}
	p.sc = parseutil.NewScanner()
	return p
}
func (p *varRefParser) parseVarRefs(src []byte) ([]*VarRef, error) {
	p.sc.SetSrc(src)
	w := []*VarRef{}
	for {
		if err := p.sc.M.EscapeAny(osutil.EscapeRune); err == nil {
			continue
		}
		if err := p.sc.M.QuotedString2('\\', 3000, 3000); err == nil {
			continue
		}
		if v, err := p.parseVarRef(); err == nil {
			w = append(w, v)
			continue
		}
		// consume rune
		if _, err := p.sc.ReadRune(); err != nil {
			break
		}
	}
	return w, nil
}
func (p *varRefParser) parseVarRef() (*VarRef, error) {
	pos0 := p.sc.KeepPos()
	vr := &VarRef{}
	err := p.sc.M.RestorePosOnErr(func() error {
		// symbol
		if err := p.sc.M.RuneAny([]rune("~$")); err != nil {
			return err
		}
		sym := p.sc.BytesFrom(pos0.Pos)
		// open/close
		hasOpen := false
		if err := p.sc.M.Rune('{'); err == nil {
			hasOpen = true
		}
		// name
		pos2 := p.sc.KeepPos()
		u := "[a-zA-Z0-9_]+"
		if err := p.sc.M.RegexpFromStartCached(u); err != nil {
			return err
		}
		name := p.sc.BytesFrom(pos2.Pos)
		// open/close
		if hasOpen {
			if err := p.sc.M.Rune('}'); err != nil {
				return err
			}
		}
		vr.Name = fmt.Sprintf("%s%s", sym, name)
		return nil
	})
	if err != nil {
		return nil, err
	}
	vr.SetPos(pos0.Pos, p.sc.Pos)
	return vr, nil
}
