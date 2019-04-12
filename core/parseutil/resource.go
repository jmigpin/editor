package parseutil

import (
	"os"
	"strings"
	"unicode"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/statemach"
)

type Resource struct {
	Path         string
	RawPath      string
	Line, Column int

	ExpandedMin, ExpandedMax int
}

func ParseResourceStr(str string, index int) (*Resource, error) {
	rw := iorw.NewBytesReadWriter([]byte(str))
	return ParseResource(rw, index)
}

func ParseResource(rd iorw.Reader, index int) (*Resource, error) {
	escape := osutil.EscapeRune

	l, r := ExpandResourceIndexes(rd, index, escape)

	res := &Resource{ExpandedMin: l, ExpandedMax: r}

	p := &ResParser{res: res, escape: escape, pathSep: os.PathSeparator}
	rd2 := iorw.NewLimitedReader(rd, l, r, 0)
	err := p.start(rd2)
	if err != nil {
		return nil, err
	}
	return res, nil
}

//----------

func ExpandResourceIndexes(rd iorw.Reader, index int, escape rune) (int, int) {
	// ensure the index is not in the middle of an escape
	index = ImproveExpandIndexEscape(rd, index, escape)

	fn := isResourceRuneNoSpace
	l := ExpandLastIndexEscape(rd, index, false, fn, escape)
	r := ExpandIndexEscape(rd, index, false, fn, escape)
	return l, r
}

//----------

type ResParser struct {
	sc  *statemach.Scanner
	st  func() bool // state
	err error

	escape  rune
	pathSep rune

	res *Resource
}

func (p *ResParser) start(r iorw.Reader) error {
	p.sc = statemach.NewScanner(r)
	// state loop
	p.st = p.pathHeader
	for {
		if p.st == nil || !p.st() {
			break
		}
	}
	return p.err
}

//----------

func (p *ResParser) pathHeader() bool {
	if !p.path() {
		p.err = p.sc.Errorf("path")
		return false
	}
	_ = p.lineCol()
	p.st = nil
	return true
}

func (p *ResParser) path() bool {
	ok := p.sc.RewindOnFalse(func() bool {
		_ = p.pathItem() // optional
		pathSepFn := func(ru rune) bool { return ru == p.pathSep }
		for {
			if p.sc.Match.End() {
				break
			}
			if !p.sc.Match.FnLoop(pathSepFn) { // any number of pathsep
				break
			}
			if !p.pathItem() {
				break
			}
		}
		return !p.sc.Empty()
	})
	if ok {
		s := p.sc.Value()
		p.res.RawPath = s

		// filter
		s = RemoveEscapes(s, p.escape)
		s = CleanPathSepSequences(s, p.pathSep)
		p.res.Path = s

		p.sc.Advance()
		return true
	}
	return false
}

func (p *ResParser) pathItem() bool {
	return p.sc.RewindOnFalse(func() bool {
		isPathItemRune := isPathItemRuneFn(p.escape)
		for p.sc.Match.Escape(p.escape) ||
			p.sc.Match.Fn(isPathItemRune) {
		}
		return !p.sc.Empty()
	})
}

//----------

func (p *ResParser) lineCol() bool {
	return p.sc.RewindOnFalse(func() bool {
		// line sep
		if !p.sc.Match.Rune(':') {
			return false
		}
		p.sc.Advance()
		// line
		v, err := p.sc.Match.IntValueAdvance()
		if err != nil {
			return false
		}
		p.res.Line = v

		_ = p.sc.RewindOnFalse(func() bool {
			// column sep
			if !p.sc.Match.Rune(':') {
				return false
			}
			p.sc.Advance()
			// column
			v, err = p.sc.Match.IntValueAdvance()
			if err != nil {
				return false
			}
			p.res.Column = v
			return true
		})

		return true
	})
}

//----------

func AddEscapes(str string, escape rune, escapeSyms string) string {
	w := []rune{}
	for _, ru := range str {
		if strings.ContainsRune(escapeSyms, ru) {
			w = append(w, escape)
		}
		w = append(w, ru)
	}
	return string(w)
}

func RemoveEscapes(str string, escape rune) string {
	w := []rune{}
	esc := false
	for _, ru := range str {
		if !esc {
			if ru == escape {
				esc = true
				continue
			}
		} else {
			esc = false
		}
		w = append(w, ru)
	}
	return string(w)
}

//----------

func CleanPathSepSequences(str string, sep rune) string {
	w := []rune{}
	added := false
	for _, ru := range str {
		if ru == sep {
			if !added {
				added = true
				w = append(w, ru)
			}
		} else {
			added = false
			w = append(w, ru)
		}
	}
	return string(w)
}

//----------

func isResourceRuneNoSpace(ru rune) bool {
	// not present: {space} must be escaped
	return unicode.IsLetter(ru) || unicode.IsDigit(ru) ||
		strings.ContainsRune(`\\`+`_-/~.%^@:&?=#`, ru)
}

func isPathItemRuneFn(escape rune) func(ru rune) bool {
	// not present (must be escaped)
	//	space (word sep)
	//	: (line/col)
	//  	()[]<> (usually used around filenames in various outputs)
	// 	\ and ^ (possible escape runes)
	extra := string(extraNonEscapeRunes(escape))
	return func(ru rune) bool {
		return unicode.IsLetter(ru) || unicode.IsDigit(ru) ||
			strings.ContainsRune(`_-/~.%@&?=#`+extra, ru)
	}
}

func extraNonEscapeRunes(escape rune) []rune {
	possibleEscapes := "\\^" // possible escape runes
	w := []rune{}
	for _, ru := range possibleEscapes {
		if ru != escape {
			w = append(w, ru)
		}
	}
	return w
}

//----------

func ImproveExpandIndexEscape(r iorw.Reader, i int, escape rune) int {
	sc := statemach.NewScanner(r)
	sc.Pos = i
	sc.RevertReadDirection()
	for {
		if sc.Match.End() {
			break
		}
		if sc.Match.Rune(escape) {
			continue
		}
		break
	}
	return sc.Pos
}

//----------

func ExpandIndexEscape(r iorw.Reader, i int, truth bool, fn func(rune) bool, escape rune) int {
	sc := statemach.NewScanner(r)
	sc.Pos = i
	return expandEscape(sc, truth, fn, escape)
}

func ExpandLastIndexEscape(r iorw.Reader, i int, truth bool, fn func(rune) bool, escape rune) int {
	sc := statemach.NewScanner(r)
	sc.Pos = i
	sc.RevertReadDirection()
	return expandEscape(sc, truth, fn, escape)
}

func expandEscape(sc *statemach.Scanner, truth bool, fn func(rune) bool, escape rune) int {
	for {
		if sc.Match.End() {
			break
		}
		if sc.Match.Escape(escape) {
			continue
		}
		u := sc.Pos
		ru := sc.ReadRune()
		if fn(ru) == truth {
			sc.Pos = u
			break
		}
	}
	return sc.Pos
}

//----------
