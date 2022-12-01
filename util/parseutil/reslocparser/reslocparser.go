package reslocparser

import (
	"sync"

	"github.com/jmigpin/editor/util/parseutil"
)

type ResLocParser struct {
	parseMu sync.Mutex // allow .Parse() to be used concurrently

	Escape        rune
	PathSeparator rune
	ParseVolume   bool

	sc *parseutil.Scanner
	fn struct {
		location ScFn
		reverse  ScFn
	}
	vk struct {
		path   *parseutil.ScValueKeeper
		line   *parseutil.ScValueKeeper
		column *parseutil.ScValueKeeper
	}
}

func NewResLocParser() *ResLocParser {
	p := &ResLocParser{}
	p.sc = parseutil.NewScanner()

	p.Escape = '\\'
	p.PathSeparator = '/'
	p.ParseVolume = false

	return p
}
func (p *ResLocParser) Init() {
	sc := p.sc

	p.vk.path = sc.NewValueKeeper()
	p.vk.line = sc.NewValueKeeper()
	p.vk.column = sc.NewValueKeeper()
	resetVks := func() error {
		p.vk.path.Reset()
		p.vk.line.Reset()
		p.vk.column.Reset()
		return nil
	}

	//----------

	nameSyms := func(pathSep, esc rune) ScFn {
		rs := nameRunes(pathSep, esc)
		return sc.P.RuneAny(rs)
	}

	volume := func(pathSepFn ScFn) ScFn {
		if p.ParseVolume {
			return sc.P.And(sc.M.Letter, sc.P.Rune(':'), pathSepFn)
		} else {
			return nil
		}
	}

	//----------

	cEscRu := p.Escape
	cPathSepRu := p.PathSeparator
	cPathSep := sc.P.Rune(cPathSepRu)
	cName := sc.P.Or(
		sc.P.EscapeAny(cEscRu),
		sc.M.Digit,
		sc.M.Letter,
		nameSyms(cPathSepRu, cEscRu),
	)
	cNames := sc.P.Loop2(sc.P.Or(
		cName,
		cPathSep,
	))
	cVolume := volume(cPathSep)
	cPath := sc.P.And(
		sc.P.Optional(cVolume),
		cNames,
	)
	cLineCol := sc.P.And(
		sc.P.Rune(':'),
		p.vk.line.KeepBytes(sc.M.Digits), // line
		sc.P.Optional(sc.P.And(
			sc.P.Rune(':'),
			p.vk.column.KeepBytes(sc.M.Digits), // column
		)),
	)
	cFile := sc.P.And(
		p.vk.path.KeepBytes(cPath),
		sc.P.Optional(cLineCol),
	)

	//----------

	schEscRu := '\\'    // fixed
	schPathSepRu := '/' // fixed
	schPathSep := sc.P.Rune(schPathSepRu)
	schName := sc.P.Or(
		sc.P.EscapeAny(schEscRu),
		sc.M.Digit,
		sc.M.Letter,
		nameSyms(schPathSepRu, schEscRu), // fixed
	)
	schNames := sc.P.Loop2(sc.P.Or(
		schName,
		schPathSep,
	))
	schVolume := volume(schPathSep)
	schPath := sc.P.And(
		schPathSep,
		sc.P.Optional(schVolume),
		schNames,
	)
	schFileTagS := "file://"
	schFile := sc.P.And(
		sc.P.Sequence(schFileTagS), // protocol
		p.vk.path.KeepBytes(schPath),
		sc.P.Optional(cLineCol),
	)

	//----------

	pyQuote := sc.P.Rune('"')
	pyLineTagS := ", line "
	pyFile := sc.P.And(
		pyQuote,
		p.vk.path.KeepBytes(cPath),
		pyQuote,
		sc.P.Optional(
			sc.P.And(
				sc.P.Sequence(pyLineTagS),
				p.vk.line.KeepBytes(sc.M.Digits),
			),
		),
	)

	//----------

	p.fn.location = sc.P.Or(
		// ensure values are reset at each attempt
		sc.P.And(resetVks, schFile),
		sc.P.And(resetVks, pyFile),
		sc.P.And(resetVks, cFile),
	)

	//----------
	//----------
	//----------

	revNames := sc.P.Loop2(
		sc.P.Or(
			// TODO: escapeany?
			cName,
			schName,
			sc.P.Rune(cEscRu),
			sc.P.Rune(schEscRu),
			cPathSep,
			schPathSep,
		),
	)
	p.fn.reverse = sc.P.And(
		sc.P.Optional(pyQuote),
		//sc.P.Optional(cVolume),
		//sc.P.Optional(schVolume),
		sc.P.Optional(sc.P.And(sc.M.Letter, sc.P.Rune(':'))), // volume
		sc.P.Optional(sc.P.SequenceMid(schFileTagS)),
		sc.P.Optional(revNames),
		sc.P.Optional(pyQuote),
		sc.P.Optional(sc.P.SequenceMid(pyLineTagS)),
		// c line column
		sc.P.Optional(sc.P.Loop2(
			sc.P.Or(sc.P.Rune(':'), sc.M.Digit),
		)),
	)
}
func (p *ResLocParser) Parse(src []byte, index int) (*ResLoc, error) {
	// only one instance of this parser can parse at each time
	p.parseMu.Lock()
	defer p.parseMu.Unlock()

	p.sc.SetSrc(src)
	p.sc.Pos = index

	//fmt.Printf("start pos=%v\n", p.sc.Pos)

	p.sc.Reverse = true
	_ = p.fn.reverse() // best effort
	p.sc.Reverse = false

	//fmt.Printf("reverse pos=%v\n", p.sc.Pos)

	//p.sc.Debug = true
	pos0 := p.sc.KeepPos()
	if err := p.fn.location(); err != nil {
		return nil, err
	}
	//fmt.Printf("location pos=%v\n", p.sc.Pos)

	rl := &ResLoc{}
	rl.Pos = pos0.Pos
	rl.End = p.sc.Pos
	rl.Escape = p.Escape
	rl.PathSep = p.PathSeparator
	rl.Path = string(p.vk.path.BytesOrNil())
	rl.Line = p.vk.line.IntOrZero()
	rl.Column = p.vk.column.IntOrZero()

	return rl, nil
}

//----------
//----------
//----------

type ScFn = parseutil.ScFn

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

func nameRunes(pathSep, esc rune) []rune {
	out := nameSepSyms + string(esc) + string(pathSep)
	s := parseutil.RunesExcept(syms, out)
	return []rune(s)
}
