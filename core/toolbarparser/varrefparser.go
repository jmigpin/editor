package toolbarparser

import (
	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil/btparser"
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

const varRefsDataKey = "toolbarparser.varrefs"
const varRefDataKey = "toolbarparser.varref"

//----------
//----------
//----------

type varRefParser struct {
	g  btparser.Rules
	fn btparser.MFn
}

func newVarRefParser() *varRefParser {
	p := &varRefParser{}
	p.g = btparser.NewRules()
	p.build()
	return p
}

func (p *varRefParser) parseVarRefs(src []byte) ([]*VarRef, error) {
	vrs := []*VarRef{}

	ps := btparser.NewParserStateFromBytes(src)
	ps.UserData[varRefsDataKey] = &vrs

	if _, err := p.g.Parse(ps, p.fn); err != nil {
		return nil, err
	}
	return vrs, nil
}

func (p *varRefParser) build() {
	g := p.g

	type varRefData struct {
		sym  string
		name string
	}

	varRefsData := btparser.UserDataPtrFn[[]*VarRef](varRefsDataKey)
	varRefPtr := btparser.UserDataPtrFn[varRefData](varRefDataKey)

	symDst := func(ps *btparser.ParserState) *string {
		return &varRefPtr(ps).sym
	}
	nameDst := func(ps *btparser.ParserState) *string {
		return &varRefPtr(ps).name
	}
	assignSym := func(fn btparser.MFn) btparser.MFn {
		return btparser.AssignFn(symDst, g.VString(fn))
	}
	assignName := func(fn btparser.MFn) btparser.MFn {
		return btparser.AssignFn(nameDst, g.VString(fn))
	}
	appendVarRef := func(fn btparser.MFn) btparser.MFn {
		return func(ps *btparser.ParserState, pos btparser.Pos) (btparser.MPos, error) {
			vrd := &varRefData{}
			ps.UserData[varRefDataKey] = vrd
			defer delete(ps.UserData, varRefDataKey)

			return btparser.AppendFn(varRefsData, btparser.VFromMPos(fn, func(ps *btparser.ParserState, mp btparser.MPos) *VarRef {
				vr := &VarRef{Name: vrd.sym + vrd.name}
				vr.SetPos(int(mp.Start), int(mp.End))
				return vr
			}))(ps, pos)
		}
	}

	//----------

	name := g.Loop1(g.Or(
		g.Rune('_'),
		g.AsciiLetter(),
		g.Digit(),
	))
	varRefFn := appendVarRef(g.And(
		assignSym(g.RuneAnyOf('~', '$')),
		g.Or(
			g.And(
				g.Rune('{'),
				assignName(name),
				g.Rune('}'),
			),
			assignName(name),
		),
	))

	p.fn = g.And(
		g.Loop1(g.Or(
			g.Escape(osutil.EscapeRune),
			g.QuotedString2('\\', 3000, 3000),
			varRefFn,
			g.AnyRune(),
		)),
		g.Eof(),
	)
}
