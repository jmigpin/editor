package reslocparser

import "github.com/jmigpin/editor/util/parseutil/btparser"

var fileSchemeTag = "file://"
var pythonLineTailTag = ", line "
var shellLineTailTag = ": line "

//----------

type ResLocParser2 struct {
	g btparser.Rules
	//scn *coverIndexScan
	rs *ReverseScan
	fn btparser.MFn
}

func NewResLocParser2(escape, pathSeparator rune, parseVolume bool) *ResLocParser2 {
	p := &ResLocParser2{}
	p.g = btparser.NewRules()

	g := p.g

	resLoc := func(ps *btparser.ParserState) *ResLoc {
		rl, ok := ps.UserData.(*ResLoc)
		if !ok || rl == nil {
			panic("missing *ResLoc in ParserState.UserData")
		}
		return rl
	}

	assignValue := func(dst func(*btparser.ParserState) *string, fn btparser.VFn[string]) btparser.MFn {
		return func(ps *btparser.ParserState, pos btparser.Pos) (btparser.MPos, error) {
			return btparser.Assign(dst(ps), fn)(ps, pos)
		}
	}
	assignInt := func(dst func(*btparser.ParserState) *int, fn btparser.VFn[int]) btparser.MFn {
		return func(ps *btparser.ParserState, pos btparser.Pos) (btparser.MPos, error) {
			return btparser.Assign(dst(ps), fn)(ps, pos)
		}
	}

	nameSyms := func(except ...rune) btparser.MFn {
		rs := buildPathItemSyms(except...)
		return g.RuneAnyOf(rs...)
	}

	volume := func(pathSepFn btparser.MFn) btparser.MFn {
		if parseVolume {
			return g.And(
				assignValue(func(ps *btparser.ParserState) *string { return &resLoc(ps).Volume }, g.VSourceStr(g.And(
					g.Letter(),
					g.Rune(':'),
				))),
				pathSepFn,
			)
		}
		return g.Fail()
	}

	cEscRu := escape
	cPathSepRu := pathSeparator
	cPathSep := g.Rune(cPathSepRu)
	cName := g.Or(
		g.Escape(cEscRu),
		g.Digit(),
		g.Letter(),
		nameSyms(cPathSepRu, cEscRu),
	)
	cNames := g.Loop1(g.Or(
		cName,
		cPathSep,
	))
	cPath := g.And(
		g.Optional(volume(cPathSep)),
		cNames,
	)
	cLineCol := g.And(
		g.Rune(':'),
		assignInt(func(ps *btparser.ParserState) *int { return &resLoc(ps).Line }, g.VInteger()),
		g.Optional(g.And(
			g.Rune(':'),
			assignInt(func(ps *btparser.ParserState) *int { return &resLoc(ps).Column }, g.VInteger()),
		)),
	)
	cOffset := g.And(
		g.Seq(":o="),
		assignInt(func(ps *btparser.ParserState) *int { return &resLoc(ps).Offset }, g.VInteger()),
	)
	cFile := g.And(
		assignValue(func(ps *btparser.ParserState) *string { return &resLoc(ps).Path }, g.VSourceStr(cPath)),
		g.Optional(g.Or(cOffset, cLineCol)),
	)

	schEscRu := '\\'
	schPathSepRu := '/'
	schPathSep := g.Rune(schPathSepRu)
	schName := g.Or(
		g.Escape(schEscRu),
		g.Digit(),
		g.Letter(),
		nameSyms(schPathSepRu, schEscRu),
	)
	schNames := g.Loop1(g.Or(
		schName,
		schPathSep,
	))
	schPath := g.And(
		schPathSep,
		g.Optional(volume(schPathSep)),
		schNames,
	)
	schFile := g.And(
		assignValue(func(ps *btparser.ParserState) *string { return &resLoc(ps).Scheme }, g.VSourceStr(g.Seq(fileSchemeTag))),
		assignValue(func(ps *btparser.ParserState) *string { return &resLoc(ps).Path }, g.VSourceStr(schPath)),
		g.Optional(cLineCol),
	)

	cPathQuoted := func(q rune) btparser.MFn {
		return g.And(
			g.Optional(volume(cPathSep)),
			g.Loop1(g.Or(
				g.Escape(cEscRu),
				g.RuneFn(func(ru rune) bool {
					return ru != q && ru != '\n'
				}),
			)),
		)
	}

	quotedFile := func(q rune) (btparser.MFn, btparser.MFn) {
		qFn := g.Rune(q)
		file := g.And(
			qFn,
			assignValue(func(ps *btparser.ParserState) *string { return &resLoc(ps).Path }, g.VSourceStr(cPathQuoted(q))),
			qFn,
		)
		fileLineCol := g.And(file, g.Optional(cLineCol))
		return file, fileLineCol
	}

	dquotedFileBase, dquotedFile := quotedFile('"')
	_, squotedFile := quotedFile('\'')
	_, bquotedFile := quotedFile('`')

	pyFile := g.And(
		dquotedFileBase,
		g.And(
			g.Seq(pythonLineTailTag),
			assignInt(func(ps *btparser.ParserState) *int { return &resLoc(ps).Line }, g.VInteger()),
		),
	)

	shellFile := g.And(
		assignValue(func(ps *btparser.ParserState) *string { return &resLoc(ps).Path }, g.VSourceStr(cPath)),
		g.And(
			g.Seq(shellLineTailTag),
			assignInt(func(ps *btparser.ParserState) *int { return &resLoc(ps).Line }, g.VInteger()),
		),
	)

	withResLocCopy := func(fn btparser.MFn) btparser.MFn {
		return func(ps *btparser.ParserState, pos btparser.Pos) (btparser.MPos, error) {
			rl1 := resLoc(ps)
			rl2 := *rl1
			ps.UserData = &rl2
			defer func() { ps.UserData = rl1 }()

			mp, err := fn(ps, pos)
			if err != nil {
				return mp, err
			}

			*rl1 = rl2
			return mp, nil
		}
	}

	p.fn = g.Or(
		withResLocCopy(schFile),
		withResLocCopy(pyFile),
		withResLocCopy(dquotedFile),
		withResLocCopy(squotedFile),
		withResLocCopy(bquotedFile),
		withResLocCopy(shellFile),
		withResLocCopy(cFile),

		// TODO: git diff
	)

	p.rs = NewReverseScanResLoc(escape, pathSeparator)
	//p.scn = newCoverIndexScan(p.g, p.fn)

	return p
}

func (p *ResLocParser2) Parse(src []byte, index int) (*ResLoc, error) {
	lineStart := index
	for lineStart > 0 && src[lineStart-1] != '\n' {
		lineStart--
	}

	rl := NewResLoc()
	rl.Escape = p.rs.escape
	rl.PathSep = p.rs.pathSep

	//----------

	p2, err := p.rs.ParseStart(src, index, 2000)
	if err != nil {
		p2 = lineStart
	}

	ps := btparser.NewParserStateFromBytes(src)
	ps.UserData = rl
	p3, err := p.g.ParseAt(ps, btparser.Pos(p2), p.fn)
	if err != nil {
		return nil, err
	}
	if int(p3) < index {
		return nil, btparser.NoMatchErr
	}

	//----------

	//p2, p3, err := p.scn.parse(src, lineStart, index, rl)
	//if err != nil {
	//	return nil, err
	//}

	//----------

	rl.Pos = p2
	rl.End = int(p3)
	return rl, nil
}
