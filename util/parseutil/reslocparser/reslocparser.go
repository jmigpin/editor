package reslocparser

import (
	"strings"
	"unicode"

	"github.com/jmigpin/editor/util/parseutil"
	"github.com/jmigpin/editor/util/parseutil/btparser"
)

const resLocDataKey = "reslocparser.resloc"
const gitDiffPathKey = "reslocparser.gitdiff.path"

var fileSchemeTag = "file://"
var pythonLineTailTag = ", line "
var shellLineTailTag = ": line "

//----------

// all syms except letters and digits
var syms = "_-~.%@&?!=#+:^(){}[]<>\\/ "

// path separator symbols
var pathSepSyms = "" +
	" " + // word separator
	"=" + // usually around filenames (ex: -arg=/a/b.txt)
	"(){}[]<>" + // usually used around filenames in various outputs
	":" + // usually separating lines/cols from filenames
	""

//----------
//----------
//----------

type ResLocParser struct {
	g       btparser.Rules
	revScan *ReverseScan
	fn      btparser.MFn
}

func NewResLocParser(escape, pathSeparator rune, parseVolume bool) *ResLocParser {
	p := &ResLocParser{}
	p.g = btparser.NewRules()
	p.init(escape, pathSeparator, parseVolume)
	return p
}

func (p *ResLocParser) Parse(src []byte, index int) (*ResLoc, error) {
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

func (p *ResLocParser) init(escape, pathSeparator rune, parseVolume bool) {
	g := p.g

	p.revScan = NewReverseScanResLoc(escape, pathSeparator, parseVolume)

	//----------

	resLocData := btparser.UserDataPtrFn[ResLoc](resLocDataKey)
	volumeDst := func(ps *btparser.ParserState) *string { return &resLocData(ps).Volume }
	schemeDst := func(ps *btparser.ParserState) *string { return &resLocData(ps).Scheme }
	pathDst := func(ps *btparser.ParserState) *string { return &resLocData(ps).Path }
	lineDst := func(ps *btparser.ParserState) *int { return &resLocData(ps).Line }
	columnDst := func(ps *btparser.ParserState) *int { return &resLocData(ps).Column }
	offsetDst := func(ps *btparser.ParserState) *int { return &resLocData(ps).Offset }

	assignVolume := func(fn btparser.MFn) btparser.MFn {
		return btparser.AssignFn(volumeDst, g.VString(fn))
	}
	assignScheme := func(fn btparser.MFn) btparser.MFn {
		return btparser.AssignFn(schemeDst, g.VString(fn))
	}
	assignPath := func(fn btparser.MFn) btparser.MFn {
		return btparser.AssignFn(pathDst, g.VString(fn))
	}
	assignLine := func(fn btparser.VFn[int]) btparser.MFn {
		return btparser.AssignFn(lineDst, fn)
	}
	assignColumn := func(fn btparser.VFn[int]) btparser.MFn {
		return btparser.AssignFn(columnDst, fn)
	}
	assignOffset := func(fn btparser.VFn[int]) btparser.MFn {
		return btparser.AssignFn(offsetDst, fn)
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

	files := g.Or(
		withResLocCopy(schFile),
		withResLocCopy(pyFile),
		withResLocCopy(dquotedFile),
		withResLocCopy(squotedFile),
		withResLocCopy(bquotedFile),
		withResLocCopy(shellFile),
		withResLocCopy(cFile),
	)

	p.fn = g.Or(
		withResLocCopy(p.buildGitDiff()),

		//// go backwards to compensate middle positions
		//g.And(
		//	p.revScan.Rule(3000),
		//	files,
		//),

		// revscan alternative (simpler backward func + brute attempts)
		g.BruteCoverPosEnd(
			g.WithBounds(3000, 0,
				g.And(
					g.ToLastIndexByteOrStart('\n'),
					g.Optional(g.Rune('\n')),
				),
			),
			files,
		),
	)
}

//----------

func (p *ResLocParser) buildGitDiff() btparser.MFn {
	g := p.g

	resLocData := btparser.UserDataPtrFn[ResLoc](resLocDataKey)
	setGitPath := func(path string, ps *btparser.ParserState) bool {
		path = strings.TrimSpace(path)
		if path == "" || path == "/dev/null" {
			return false
		}
		resLocData(ps).Path = trimGitDiffPathPrefix(path)
		return true
	}
	gitDiffPathData := btparser.UserDataPtrFn[string](gitDiffPathKey)
	lineDst := func(ps *btparser.ParserState) *int { return &resLocData(ps).Line }
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
		return btparser.AssignFn(lineDst, fn)
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
		g.BruteCoverPos(
			toLineStart,
			g.And(
				g.Seq("@@"),
				g.Spaces(),
				gitDiffOldRange,
				g.Spaces(),
				gitDiffNewRange,
				g.Spaces(),
				g.Seq("@@"),
			),
		),

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

//----------

func buildPathItemSyms(except ...rune) []rune {
	out := pathSepSyms
	for _, ru := range except {
		if ru != 0 {
			out += string(ru)
		}
	}
	s := parseutil.RunesExcept(syms, out)
	return []rune(s)
}
