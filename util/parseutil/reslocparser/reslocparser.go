package reslocparser

import (
	"sync"

	"github.com/jmigpin/editor/util/parseutil"
	"github.com/jmigpin/editor/util/parseutil/pscan"
)

type MFn = pscan.MFn

//----------

type ResLocParser struct {
	parseMu sync.Mutex // allow .Parse() to be used concurrently

	Escape        rune
	PathSeparator rune
	ParseVolume   bool

	sc *pscan.Scanner
	fn struct {
		location MFn
		reverse  MFn
	}
	vk struct {
		scheme *string
		volume *string
		path   *string
		line   *int
		column *int
		offset *int
	}
}

func NewResLocParser() *ResLocParser {
	p := &ResLocParser{}
	p.sc = pscan.NewScanner()

	p.Escape = '\\'
	p.PathSeparator = '/'
	p.ParseVolume = false

	return p
}
func (p *ResLocParser) Init() {
	sc := p.sc

	resetVks := func(pos int) (int, error) {
		p.vk.scheme = nil
		p.vk.volume = nil
		p.vk.path = nil
		p.vk.line = nil
		p.vk.column = nil
		p.vk.offset = nil
		return pos, nil
	}

	//----------

	nameSyms := func(except ...rune) MFn {
		rs := nameRunes(except...)
		return sc.W.RuneOneOf(rs)
	}

	volume := func(pathSepFn MFn) MFn {
		if p.ParseVolume {
			return sc.W.And(
				pscan.WKeep(&p.vk.volume, sc.W.StrValue(sc.W.And(
					sc.M.Letter,
					sc.W.Rune(':'),
				))),
				pathSepFn,
			)
		} else {
			return func(pos int) (int, error) {
				return pos, pscan.NoMatchErr
			}
		}
	}

	//----------

	// ex: "/a/b.txt"
	// ex: "/a/b.txt:12:3"
	// ex: "/a/b.txt:o=123" // offset (custom format)
	cEscRu := p.Escape
	cPathSepRu := p.PathSeparator
	cPathSep := sc.W.Rune(cPathSepRu)
	cName := sc.W.Or(
		sc.W.EscapeAny(cEscRu),
		sc.M.Digit,
		sc.M.Letter,
		nameSyms(cPathSepRu, cEscRu),
	)
	cNames := sc.W.LoopOneOrMore(sc.W.Or(
		cName,
		cPathSep,
	))
	cPath := sc.W.And(
		sc.W.Optional(volume(cPathSep)),
		cNames,
	)
	cLineCol := sc.W.And(
		sc.W.Rune(':'),
		pscan.WKeep(&p.vk.line, sc.M.IntValue), // line
		sc.W.Optional(sc.W.And(
			sc.W.Rune(':'),
			pscan.WKeep(&p.vk.column, sc.M.IntValue), // column
		)),
	)
	cOffset := sc.W.And(
		sc.W.Sequence(":o="),
		pscan.WKeep(&p.vk.offset, sc.M.IntValue),
	)
	cFile := sc.W.And(
		pscan.WKeep(&p.vk.path, sc.W.StrValue(cPath)),
		sc.W.Optional(sc.W.Or(cOffset, cLineCol)),
	)

	//----------

	// ex: "file:///a/b.txt:12"
	// no escape sequence for scheme, used to be '\\' but better to avoid conflicts with platforms that use '\\' as escape; could always use encoding (ex: %20 for ' ')
	schEscRu := '\\'    // fixed
	schPathSepRu := '/' // fixed
	schPathSep := sc.W.Rune(schPathSepRu)
	schName := sc.W.Or(
		sc.W.EscapeAny(schEscRu),
		sc.M.Digit,
		sc.M.Letter,
		nameSyms(schPathSepRu, schEscRu),
	)
	schNames := sc.W.LoopOneOrMore(sc.W.Or(
		schName,
		schPathSep,
	))
	schPath := sc.W.And(
		schPathSep,
		sc.W.Optional(volume(schPathSep)),
		schNames,
	)
	schFileTagS := "file://"
	schFile := sc.W.And(
		pscan.WKeep(&p.vk.scheme, sc.W.StrValue(
			sc.W.Sequence(schFileTagS)),
		),
		pscan.WKeep(&p.vk.path, sc.W.StrValue(schPath)),
		sc.W.Optional(cLineCol),
	)

	//----------

	// ex: "\"/a/b.txt\""
	dquote := sc.W.Rune('"') // double quote
	dquotedFile := sc.W.And(
		dquote,
		pscan.WKeep(&p.vk.path, sc.W.StrValue(cPath)),
		dquote,
	)

	//----------

	// ex: "\"/a/b.txt\", line 23"
	pyLineTagS := ", line "
	pyFile := sc.W.And(
		dquotedFile,
		sc.W.And(
			sc.W.Sequence(pyLineTagS),
			pscan.WKeep(&p.vk.line, sc.M.IntValue),
		),
	)

	//----------

	// ex: "/a/b.txt: line 23"
	shellLineTagS := ": line "
	shellFile := sc.W.And(
		pscan.WKeep(&p.vk.path, sc.W.StrValue(cPath)),
		sc.W.And(
			sc.W.Sequence(shellLineTagS),
			pscan.WKeep(&p.vk.line, sc.M.IntValue),
		),
	)

	//----------

	p.fn.location = sc.W.Or(
		// ensure values are reset at each attempt
		sc.W.And(resetVks, schFile),
		sc.W.And(resetVks, pyFile),
		sc.W.And(resetVks, dquotedFile),
		sc.W.And(resetVks, shellFile),
		sc.W.And(resetVks, cFile),
	)

	//----------
	//----------

	revNames := sc.W.LoopOneOrMore(
		sc.W.Or(
			cName,
			//schName, // can't reverse, contains fixed '\\' escape that can conflit with platform not considering it an escape
			sc.W.Rune(cEscRu),
			sc.W.Rune(schEscRu),
			cPathSep,
			schPathSep,
		),
	)
	p.fn.reverse = sc.W.ReverseMode(true, sc.W.And(
		sc.W.Optional(dquote),
		//sc.P.Optional(cVolume),
		//sc.P.Optional(schVolume),
		sc.W.Optional(sc.W.SequenceMid(schFileTagS)),
		sc.W.Optional(sc.W.LoopOneOrMore(sc.W.Or(
			cPathSep,
			schPathSep,
		))),
		sc.W.Optional(sc.W.And(
			sc.M.Letter,
			sc.W.Rune(':'), // volume
		)),
		sc.W.Optional(revNames),
		sc.W.Optional(dquote),
		sc.W.Optional(sc.W.SequenceMid(pyLineTagS)),
		sc.W.Optional(sc.W.SequenceMid(shellLineTagS)),
		// c line column / offset
		sc.W.Optional(sc.W.LoopOneOrMore(sc.W.Or(
			sc.W.Rune(':'),
			sc.W.RuneOneOf([]rune("o=")), // offset
			sc.M.Digit,
		))),
	))
}
func (p *ResLocParser) Parse(src []byte, index int) (*ResLoc, error) {
	// only one instance of this parser can parse at each time
	p.parseMu.Lock()
	defer p.parseMu.Unlock()

	p.sc.SetSrc(src)

	// reverse, best effort (no errors)
	p2 := index
	if u, err := p.fn.reverse(p2); err == nil {
		p2 = u
	}

	p3, err := p.fn.location(p2)
	if err != nil {
		return nil, err
	}

	rl := NewResLoc()
	if p.vk.scheme != nil {
		rl.Scheme = *p.vk.scheme
	}
	if p.vk.volume != nil {
		rl.Volume = *p.vk.volume
	}
	if p.vk.path != nil {
		rl.Path = *p.vk.path
	}
	if p.vk.line != nil {
		rl.Line = *p.vk.line
	}
	if p.vk.column != nil {
		rl.Column = *p.vk.column
	}
	if p.vk.offset != nil {
		rl.Offset = *p.vk.offset
	}
	rl.Escape = p.Escape
	rl.PathSep = p.PathSeparator
	rl.Pos = p2
	rl.End = p3

	return rl, nil
}

//----------
//----------
//----------

// all syms except letters and digits
var syms = "_-~.%@&?!=#+:^(){}[]<>\\/ "

// name separator symbols
var nameSepSyms = "" +
	" " + // word separator
	"=" + // usually around filenames (ex: -arg=/a/b.txt)
	"(){}[]<>" + // usually used around filenames in various outputs
	":" + // usually separating lines/cols from filenames
	""

func nameRunes(except ...rune) []rune {
	out := nameSepSyms
	for _, ru := range except {
		if ru != 0 {
			out += string(ru)
		}
	}
	s := parseutil.RunesExcept(syms, out)
	return []rune(s)
}
