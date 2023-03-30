package toolbarparser

import (
	"fmt"

	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil/pscan"
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
	sc *pscan.Scanner
}

func newVarRefParser() *varRefParser {
	p := &varRefParser{}
	p.sc = pscan.NewScanner()
	return p
}
func (p *varRefParser) parseVarRefs(src []byte) ([]*VarRef, error) {
	p.sc.SetSrc(src)
	vrs := []*VarRef{}
	_, err := p.sc.M.Loop(0,
		p.sc.W.Or(
			p.sc.W.EscapeAny(osutil.EscapeRune),
			p.sc.W.QuotedString2('\\', 3000, 3000),
			p.sc.W.OnValue(
				p.parseVarRef,
				func(v any) { vrs = append(vrs, v.(*VarRef)) },
			),
			p.sc.M.OneRune,
		),
	)
	return vrs, err
}
func (p *varRefParser) parseVarRef(pos int) (any, int, error) {
	symK := p.sc.NewValueKeeper()
	nameK := p.sc.NewValueKeeper()
	parseName := func(p2 int) (int, error) {
		u := "[a-zA-Z0-9_]+"
		return nameK.KeepValue(p2, p.sc.W.StringValue(p.sc.W.RegexpFromStartCached(u, 100)))
	}

	if p3, err := p.sc.M.And(pos,
		symK.WKeepValue(p.sc.W.StringValue(p.sc.W.RuneOneOf([]rune("~$")))),
		p.sc.W.Or(
			p.sc.W.And(
				p.sc.W.Rune('{'),
				parseName,
				p.sc.W.Rune('}'),
			),
			parseName,
		),
	); err != nil {
		return nil, p3, err
	} else {
		vr := &VarRef{}
		vr.Name = fmt.Sprintf("%s%s", symK.V.(string), nameK.V.(string))
		vr.SetPos(pos, p3)
		return vr, p3, nil
	}
}
