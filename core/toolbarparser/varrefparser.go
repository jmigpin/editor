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

	// SLOWER
	//vrs := []*VarRef{}
	//err := p.sc.P.Loop(
	//	p.sc.P.Or(
	//		p.sc.P.EscapeAny(osutil.EscapeRune),
	//		p.sc.P.QuotedString2('\\', 3000, 3000),
	//		func() error {
	//			vr, err := p.parseVarRef()
	//			if err == nil {
	//				vrs = append(vrs, vr)
	//			}
	//			return err
	//		},
	//		p.sc.P.NRunes(1), // consume rune
	//	),
	//	nil, false,
	//)()
	//return vrs, err

}
func (p *varRefParser) parseVarRef() (*VarRef, error) {
	pos0 := p.sc.KeepPos()
	vr := &VarRef{}
	err := p.sc.RestorePosOnErr(func() error {
		// symbol
		if err := p.sc.M.RuneAny([]rune("~$")); err != nil {
			return err
		}
		sym := pos0.Bytes()
		// open/close
		hasOpen := false
		if err := p.sc.M.Rune('{'); err == nil {
			hasOpen = true
		}
		// name
		pos2 := p.sc.KeepPos()
		u := "[a-zA-Z0-9_]+"
		if err := p.sc.M.RegexpFromStartCached(u, 100); err != nil {
			return err
		}
		name := pos2.Bytes()
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

	// SLOWER
	//pos0 := p.sc.KeepPos()
	//symK := p.sc.P.NewValueKeeper()
	//nameK := p.sc.P.NewValueKeeper()
	//parseName := func() error {
	//	u := "[a-zA-Z0-9_]+"
	//	return nameK.KeepBytes(p.sc.P.RegexpFromStartCached(u, 100))()
	//}
	//if err := p.sc.P.And(
	//	symK.KeepBytes(p.sc.P.RuneAny([]rune("~$"))),
	//	p.sc.P.Or(
	//		p.sc.P.And(
	//			p.sc.P.Rune('{'),
	//			parseName,
	//			p.sc.P.Rune('}'),
	//		),
	//		parseName,
	//	),
	//)(); err != nil {
	//	return nil, err
	//}
	//vr := &VarRef{}
	//vr.Name = fmt.Sprintf("%s%s", symK.Bytes(), nameK.Bytes())
	//vr.SetPos(pos0.Pos, p.sc.Pos)
	//return vr, nil
}
