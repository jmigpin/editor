package reslocparser

import (
	"strings"
	"unicode"

	"github.com/jmigpin/editor/util/parseutil/btparser"
)

const resLocDataKey = "reslocparser.resloc"
const gitDiffPathKey = "reslocparser.gitdiff.path"

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

	assignVolume := func(fn btparser.MFn) btparser.MFn {
		return btparser.AssignFn(
			func(ps *btparser.ParserState) *string {
				return &resLocData(ps).Volume
			},
			g.VString(fn),
		)
	}
	assignScheme := func(fn btparser.MFn) btparser.MFn {
		return btparser.AssignFn(
			func(ps *btparser.ParserState) *string {
				return &resLocData(ps).Scheme
			},
			g.VString(fn),
		)
	}
	assignPath := func(fn btparser.MFn) btparser.MFn {
		return btparser.AssignFn(
			func(ps *btparser.ParserState) *string {
				return &resLocData(ps).Path
			},
			g.VString(fn),
		)
	}
	assignLine := func(fn btparser.VFn[int]) btparser.MFn {
		return btparser.AssignFn(
			func(ps *btparser.ParserState) *int {
				return &resLocData(ps).Line
			},
			fn,
		)
	}
	assignColumn := func(fn btparser.VFn[int]) btparser.MFn {
		return btparser.AssignFn(
			func(ps *btparser.ParserState) *int {
				return &resLocData(ps).Column
			},
			fn,
		)
	}
	assignOffset := func(fn btparser.VFn[int]) btparser.MFn {
		return btparser.AssignFn(
			func(ps *btparser.ParserState) *int {
				return &resLocData(ps).Offset
			},
			fn,
		)
	}
	volume := func(pathSepFn btparser.MFn) btparser.MFn {
		return g.And(
			g.IsTrue(parseVolume),
			assignVolume(g.And(g.Letter(), g.Rune(':'))),
			pathSepFn,
		)
	}
	quotedPath := func(q rune, path btparser.MFn) btparser.MFn {
		qFn := g.Rune(q)
		return g.And(qFn, assignPath(path), qFn)
	}

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

	//----------

	nameSyms := func(except ...rune) btparser.MFn {
		rs := buildPathItemSyms(except...)
		return g.RuneAnyOf(rs...)
	}

	//----------

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
		assignLine(g.VInteger()),
		g.Optional(g.And(
			g.Rune(':'),
			assignColumn(g.VInteger()),
		)),
	)
	cOffset := g.And(
		g.Seq(":o="),
		assignOffset(g.VInteger()),
	)
	cFile := g.And(
		assignPath(cPath),
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
		assignScheme(g.Seq(fileSchemeTag)),
		assignPath(schPath),
		g.Optional(cLineCol),
	)

	cPathQuoted := func(q rune) btparser.MFn {
		return g.And(
			g.Optional(volume(cPathSep)),
			g.Loop1(g.Or(
				g.Escape(cEscRu),
				g.RuneNotAnyOf(q, '\n'),
			)),
		)
	}

	quotedFile := func(q rune) (btparser.MFn, btparser.MFn) {
		file := quotedPath(q, cPathQuoted(q))
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
			assignLine(g.VInteger()),
		),
	)

	shellFile := g.And(
		assignPath(cPath),
		g.And(
			g.Seq(shellLineTailTag),
			assignLine(g.VInteger()),
		),
	)

	p.fn = g.Or(
		withResLocCopy(p.buildGitDiff()),

		g.And(
			// go backwards to compensate middle positions
			p.revScan.Rule(3000),
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

func (p *ResLocParser2) buildGitDiff() btparser.MFn {
	g := p.g

	resLocData := func(ps *btparser.ParserState) *ResLoc {
		rl, ok := ps.UserData[resLocDataKey].(*ResLoc)
		if !ok {
			panic("resloc parser missing ResLoc userdata")
		}
		return rl
	}
	setGitPath := func(path string, ps *btparser.ParserState) bool {
		path = strings.TrimSpace(path)
		if path == "" || path == "/dev/null" {
			return false
		}
		resLocData(ps).Path = trimGitDiffPathPrefix(path)
		return true
	}
	gitDiffPathData := func(ps *btparser.ParserState) *string {
		path, ok := ps.UserData[gitDiffPathKey].(*string)
		if !ok {
			panic("git diff parser missing path userdata")
		}
		return path
	}
	withGitDiffPath := func(fn btparser.MFn) btparser.MFn {
		return func(ps *btparser.ParserState, pos btparser.Pos) (btparser.MPos, error) {
			path := ""
			ps.UserData[gitDiffPathKey] = &path
			defer delete(ps.UserData, gitDiffPathKey)
			return fn(ps, pos)
		}
	}
	setGitPathFn := func(ps *btparser.ParserState, pos btparser.Pos) (btparser.MPos, error) {
		if !setGitPath(*gitDiffPathData(ps), ps) {
			return btparser.MPos{Start: pos, End: pos}, btparser.NoMatchErr
		}
		return btparser.MPos{Start: pos, End: pos}, nil
	}
	assignGitDiffPath := func(fn btparser.MFn) btparser.MFn {
		return btparser.AssignFn(gitDiffPathData, g.VString(fn))
	}
	assignGitDiffLine := func(fn btparser.VFn[int]) btparser.MFn {
		return btparser.AssignFn(
			func(ps *btparser.ParserState) *int {
				return &resLocData(ps).Line
			},
			fn,
		)
	}

	//----------

	gitDiffRangeTail := g.Optional(g.And(g.Rune(','), g.Integer()))
	gitDiffOldRange := g.And(g.Rune('-'), g.Integer(), gitDiffRangeTail)
	gitDiffNewRange := g.And(
		g.Rune('+'),
		assignGitDiffLine(g.VInteger()),
		gitDiffRangeTail,
	)

	gitFilePath := assignGitDiffPath(g.LoopToNLOrEof(0, false))
	gitDiffPathToken := assignGitDiffPath(
		g.Loop1(g.RuneFn(func(ru rune) bool {
			return !unicode.IsSpace(ru)
		})),
	)

	plusFileLine := withGitDiffPath(g.And(
		g.Seq("+++ "),
		gitFilePath,
		setGitPathFn,
	))

	diffGitFileLine := withGitDiffPath(g.And(
		g.Seq("diff --git "),
		gitDiffPathToken,
		g.Spaces(),
		gitFilePath,
		setGitPathFn,
	))
	gitFileLine := g.Or(
		plusFileLine,
		diffGitFileLine,
	)

	toLineStart := g.And(
		g.ToLastIndexByteOrStart('\n'),
		g.Optional(g.Rune('\n')),
	)

	hunk := g.And(
		// consume in reverse to start of line searching for header
		g.Optional(g.WithBounds(1000, 0,
			g.ReverseSource(g.LoopConsumeUntil(
				g.RuneNotAnyOf('\n'),
				g.And(
					g.SeqOrMid(btparser.ReverseString("@@ ")),
					g.Or(
						g.Peek(g.Byte('\n')),
						g.Eof(),
					),
				),
			)),
		)),

		g.Seq("@@"),
		g.Spaces(),
		gitDiffOldRange,
		g.Spaces(),
		gitDiffNewRange,
		g.Spaces(),
		g.Seq("@@"),

		// consume backwards
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

//----------
//----------
//----------

func trimGitDiffPathPrefix(path string) string {
	if strings.HasPrefix(path, "a/") || strings.HasPrefix(path, "b/") {
		return path[2:]
	}
	return path
}
