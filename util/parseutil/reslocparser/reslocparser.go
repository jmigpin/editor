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
		scheme *pscan.ValueKeeper
		volume *pscan.ValueKeeper
		path   *pscan.ValueKeeper
		line   *pscan.ValueKeeper
		column *pscan.ValueKeeper
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

	p.vk.scheme = sc.NewValueKeeper()
	p.vk.volume = sc.NewValueKeeper()
	p.vk.path = sc.NewValueKeeper()
	p.vk.line = sc.NewValueKeeper()
	p.vk.column = sc.NewValueKeeper()
	resetVks := func(pos int) (int, error) {
		p.vk.scheme.V = nil
		p.vk.volume.V = nil
		p.vk.path.V = nil
		p.vk.line.V = nil
		p.vk.column.V = nil
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
				p.vk.volume.WKeepValue(sc.W.StringValue(sc.W.And(
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
	cEscRu := p.Escape
	cPathSepRu := p.PathSeparator
	cPathSep := sc.W.Rune(cPathSepRu)
	cName := sc.W.Or(
		sc.W.EscapeAny(cEscRu),
		sc.M.Digit,
		sc.M.Letter,
		nameSyms(cPathSepRu, cEscRu),
	)
	cNames := sc.W.Loop(sc.W.Or(
		cName,
		cPathSep,
	))
	cPath := sc.W.And(
		sc.W.Optional(volume(cPathSep)),
		cNames,
	)
	cLineCol := sc.W.And(
		sc.W.Rune(':'),
		p.vk.line.WKeepValue(sc.M.IntValue), // line
		sc.W.Optional(sc.W.And(
			sc.W.Rune(':'),
			p.vk.column.WKeepValue(sc.M.IntValue), // column
		)),
	)
	cFile := sc.W.And(
		p.vk.path.WKeepValue(sc.W.StringValue(cPath)),
		sc.W.Optional(cLineCol),
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
	schNames := sc.W.Loop(sc.W.Or(
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
		p.vk.scheme.WKeepValue(sc.W.StringValue(
			sc.W.Sequence(schFileTagS)),
		),
		p.vk.path.WKeepValue(sc.W.StringValue(schPath)),
		sc.W.Optional(cLineCol),
	)

	//----------

	// ex: "\"/a/b.txt\""
	dquote := sc.W.Rune('"') // double quote
	dquotedFile := sc.W.And(
		dquote,
		p.vk.path.WKeepValue(sc.W.StringValue(cPath)),
		dquote,
	)

	//----------

	// ex: "\"/a/b.txt\", line 23"
	pyLineTagS := ", line "
	pyFile := sc.W.And(
		dquotedFile,
		sc.W.And(
			sc.W.Sequence(pyLineTagS),
			p.vk.line.WKeepValue(sc.M.IntValue),
		),
	)

	//----------

	// ex: "/a/b.txt: line 23"
	shellLineTagS := ": line "
	shellFile := sc.W.And(
		p.vk.path.WKeepValue(sc.W.StringValue(cPath)),
		sc.W.And(
			sc.W.Sequence(shellLineTagS),
			p.vk.line.WKeepValue(sc.M.IntValue),
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

	revNames := sc.W.Loop(
		sc.W.Or(
			cName,
			//schName, // can't reverse, contains fixed '\\' escape that can conflit with platform not considering it an escape
			sc.W.Rune(cEscRu),
			sc.W.Rune(schEscRu),
			cPathSep,
			schPathSep,
		),
	)
	p.fn.reverse = sc.W.ReverseMode(true, sc.W.AndR(
		sc.W.Optional(dquote),
		//sc.P.Optional(cVolume),
		//sc.P.Optional(schVolume),
		sc.W.Optional(sc.W.SequenceMid(schFileTagS)),
		sc.W.Optional(sc.W.Loop(sc.W.Or(
			cPathSep,
			schPathSep,
		))),
		sc.W.Optional(sc.W.AndR(
			sc.M.Letter,
			sc.W.Rune(':'), // volume
		)),
		sc.W.Optional(revNames),
		sc.W.Optional(dquote),
		sc.W.Optional(sc.W.SequenceMid(pyLineTagS)),
		sc.W.Optional(sc.W.SequenceMid(shellLineTagS)),
		// c line column
		sc.W.Optional(sc.W.Loop(sc.W.Or(
			sc.W.Rune(':'),
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

	rl := &ResLoc{}
	if p.vk.scheme.V != nil {
		rl.Scheme = p.vk.scheme.V.(string)
	}
	if p.vk.volume.V != nil {
		rl.Volume = p.vk.volume.V.(string)
	}
	if p.vk.path.V != nil {
		rl.Path = p.vk.path.V.(string)
	}
	if p.vk.line.V != nil {
		rl.Line = p.vk.line.V.(int)
	}
	if p.vk.column.V != nil {
		rl.Column = p.vk.column.V.(int)
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
