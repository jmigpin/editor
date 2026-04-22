package reslocparser

import (
	"strings"

	"github.com/jmigpin/editor/util/parseutil/btparser"
)

const resLocDataKey = "reslocparser.resloc"
const gitDiffInitialPosKey = "reslocparser.gitdiff.initialpos"

var fileSchemeTag = "file://"
var pythonLineTailTag = ", line "
var shellLineTailTag = ": line "

//----------

type ResLocParser2 struct {
	g       btparser.Rules
	revScan *ReverseScan
	fn      btparser.MFn
}

func NewResLocParser2(escape, pathSeparator rune, parseVolume bool) *ResLocParser2 {
	p := &ResLocParser2{}
	p.g = btparser.NewRules()
	p.init(escape, pathSeparator, parseVolume)
	return p
}

func (p *ResLocParser2) Parse(src []byte, index int) (*ResLoc, error) {
	rl := NewResLoc()
	rl.Escape = p.revScan.escape
	rl.PathSep = p.revScan.pathSep

	ps := btparser.NewParserStateFromBytes(src)
	ps.UserData[resLocDataKey] = rl

	_, err := p.g.ParseAt(ps, btparser.Pos(index), p.fn)
	// resloc values are filled inside the parse
	return rl, err
}

//----------

func (p *ResLocParser2) init(escape, pathSeparator rune, parseVolume bool) {
	g := p.g

	p.revScan = NewReverseScanResLoc(escape, pathSeparator, parseVolume)

	//----------

	resLocData := func(ps *btparser.ParserState) *ResLoc {
		rl, ok := ps.UserData[resLocDataKey].(*ResLoc)
		if !ok {
			panic("resloc parser missing ResLoc userdata")
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
		return g.And(
			g.IsTrue(parseVolume),
			assignValue(
				func(ps *btparser.ParserState) *string {
					return &resLocData(ps).Volume
				},
				g.VString(g.And(g.Letter(), g.Rune(':'))),
			),
			pathSepFn,
		)
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
		assignInt(func(ps *btparser.ParserState) *int { return &resLocData(ps).Line }, g.VInteger()),
		g.Optional(g.And(
			g.Rune(':'),
			assignInt(func(ps *btparser.ParserState) *int { return &resLocData(ps).Column }, g.VInteger()),
		)),
	)
	cOffset := g.And(
		g.Seq(":o="),
		assignInt(
			func(ps *btparser.ParserState) *int {
				return &resLocData(ps).Offset
			},
			g.VInteger(),
		),
	)
	cFile := g.And(
		assignValue(func(ps *btparser.ParserState) *string { return &resLocData(ps).Path }, g.VString(cPath)),
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
		assignValue(
			func(ps *btparser.ParserState) *string {
				return &resLocData(ps).Scheme
			},
			g.VString(g.Seq(fileSchemeTag)),
		),
		assignValue(
			func(ps *btparser.ParserState) *string {
				return &resLocData(ps).Path
			},
			g.VString(schPath),
		),
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
			assignValue(
				func(ps *btparser.ParserState) *string {
					return &resLocData(ps).Path
				},
				g.VString(cPathQuoted(q)),
			),
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
			assignInt(func(ps *btparser.ParserState) *int { return &resLocData(ps).Line }, g.VInteger()),
		),
	)

	shellFile := g.And(
		assignValue(func(ps *btparser.ParserState) *string { return &resLocData(ps).Path }, g.VString(cPath)),
		g.And(
			g.Seq(shellLineTailTag),
			assignInt(func(ps *btparser.ParserState) *int { return &resLocData(ps).Line }, g.VInteger()),
		),
	)

	withResLocCopy := func(fn btparser.MFn) btparser.MFn {
		return func(ps *btparser.ParserState, pos btparser.Pos) (btparser.MPos, error) {
			rl1 := resLocData(ps)
			rl2 := *rl1
			ps.UserData[resLocDataKey] = &rl2
			defer func() { ps.UserData[resLocDataKey] = rl1 }()

			mp, err := fn(ps, pos)
			if err != nil {
				return mp, err
			}
			rl2.Pos = int(mp.Start)
			rl2.End = int(mp.End)

			*rl1 = rl2
			return mp, nil
		}
	}

	p.fn = g.Or(
		withResLocCopy(p.buildGitDiff(resLocData, assignInt)),

		g.And(
			p.revScan.Rule(800),
			//coverIndex(index, p.fn), // TODO: revscan alternative

			g.Or(
				withResLocCopy(schFile),
				withResLocCopy(pyFile),
				withResLocCopy(dquotedFile),
				withResLocCopy(squotedFile),
				withResLocCopy(bquotedFile),
				withResLocCopy(shellFile),
				withResLocCopy(cFile),
			),
		),
	)
}

//----------

func (p *ResLocParser2) buildGitDiff(
	resLocData func(*btparser.ParserState) *ResLoc,
	assignInt func(
		func(*btparser.ParserState) *int,
		btparser.VFn[int],
	) btparser.MFn) btparser.MFn {
	g := p.g

	gitDiffRangeTail := g.Optional(g.And(g.Rune(','), g.Integer()))
	gitDiffOldRange := g.And(g.Rune('-'), g.Integer(), gitDiffRangeTail)
	gitDiffNewRange := g.And(
		g.Rune('+'),
		assignInt(
			func(ps *btparser.ParserState) *int {
				return &resLocData(ps).Line
			},
			g.VInteger(),
		),
		gitDiffRangeTail,
	)

	setGitPath := func(path string, ps *btparser.ParserState) bool {
		path = strings.TrimSpace(path)
		if path == "" || path == "/dev/null" {
			return false
		}
		resLocData(ps).Path = trimGitDiffPathPrefix(path)
		return true
	}

	setGitPathFn := func(path *string) btparser.MFn {
		return func(ps *btparser.ParserState, pos btparser.Pos) (btparser.MPos, error) {
			if !setGitPath(*path, ps) {
				return btparser.MPos{Start: pos, End: pos}, btparser.NoMatchErr
			}
			return btparser.MPos{Start: pos, End: pos}, nil
		}
	}

	gitFilePath := func(path *string) btparser.MFn {
		return btparser.Assign(path, g.VString(g.LoopToNLOrEof(0, false)))
	}

	gitDiffPathToken := func(path *string) btparser.MFn {
		return btparser.Assign(path, g.VString(g.Loop1(g.And(
			g.Not(g.Space()),
			g.AnyRune(),
		))))
	}

	plusFileLine := func(ps *btparser.ParserState, pos btparser.Pos) (btparser.MPos, error) {
		path := ""
		return g.And(
			g.Seq("+++ "),
			gitFilePath(&path),
			setGitPathFn(&path),
		)(ps, pos)
	}

	diffGitFileLine := func(ps *btparser.ParserState, pos btparser.Pos) (btparser.MPos, error) {
		oldPath := ""
		newPath := ""
		return g.And(
			g.Seq("diff --git "),
			gitDiffPathToken(&oldPath),
			g.Spaces(),
			gitFilePath(&newPath),
			setGitPathFn(&newPath),
		)(ps, pos)
	}
	gitFileLine := g.Or(
		plusFileLine,
		diffGitFileLine,
	)

	toLineStart := g.And(
		g.ToLastIndexByteOrStart('\n'),
		g.Optional(g.Rune('\n')),
	)

	hunk := g.And(
		// keep initial position for end validation
		func(ps *btparser.ParserState, pos btparser.Pos) (btparser.MPos, error) {
			ps.UserData[gitDiffInitialPosKey] = pos
			return btparser.MPos{pos, pos}, nil
		},

		toLineStart,

		g.Seq("@@"),
		g.Spaces(),
		gitDiffOldRange,
		g.Spaces(),
		gitDiffNewRange,
		g.Spaces(),
		g.Seq("@@"),

		// the initial pos must be inside the match
		func(ps *btparser.ParserState, pos btparser.Pos) (btparser.MPos, error) {
			mp := btparser.MPos{pos, pos}
			initialPos, ok := ps.UserData[gitDiffInitialPosKey].(btparser.Pos)
			if !ok {
				panic("git diff parser missing initial pos userdata")
			}
			if pos <= initialPos {
				return mp, btparser.NoMatchErr
			}
			return mp, nil
		},

		toLineStart,
		g.LoopConsumeUntil(
			g.And(
				g.LastAnyRune(), // newline
				toLineStart,
			),
			gitFileLine,
		),
	)

	return hunk
}

// ----------
// ----------
func trimGitDiffPathPrefix(path string) string {
	if strings.HasPrefix(path, "a/") || strings.HasPrefix(path, "b/") {
		return path[2:]
	}
	return path
}
