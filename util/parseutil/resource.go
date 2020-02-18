package parseutil

import (
	"os"
	"runtime"
	"strings"
	"unicode"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/scanutil"
)

var PathSeparator rune = rune(os.PathSeparator)
var Escape rune = rune(osutil.EscapeRune)
var ParseVolume bool = runtime.GOOS == "windows"

//----------

var ExtraRunes = "_-~.%@&?!=#+:^" + "(){}[]<>" + "\\/" + " "

var excludeResourceRunes = "" +
	" " + // word separator
	"=" + // usually around filenames (ex: -arg=/a/b.txt)
	"(){}[]<>" // usually used around filenames in various outputs

var ResourceExtraRunes = RunesExcept(ExtraRunes, excludeResourceRunes)

var PathExtraRunes = RunesExcept(ExtraRunes, excludeResourceRunes+
	":") // line/column

// escaped when outputing filenames
var escapedInFilenames = excludeResourceRunes +
	":" // note: in windows will give "C^:/"

//----------

// parsed formats:
// 	<filename:line?:col?>
// 	<filename:#offset> # TODO
// 	file://<filename:line?:col?> # filename should be absolute starting with "/"
type Resource struct {
	Path         string
	RawPath      string
	Line, Column int

	ExpandedMin, ExpandedMax int
	PathSep                  rune
	Escape                   rune
	ParseVolume              bool
}

func ParseResource(rd iorw.Reader, index int) (*Resource, error) {
	return ParseResource2(rd, index, PathSeparator, Escape, ParseVolume)
}

func ParseResource2(rd iorw.Reader, index int, sep, esc rune, parseVolume bool) (*Resource, error) {
	res := &Resource{
		PathSep:     sep,
		Escape:      esc,
		ParseVolume: parseVolume,
	}

	rp := &ResParser2{res: res}
	if err := rp.parse(rd, index); err != nil {
		return nil, err
	}

	return res, nil
}

//----------

type ResParser2 struct {
	res *Resource
	sc  *scanutil.Scanner
}

func (rp *ResParser2) parse(rd iorw.Reader, index int) error {
	// ensure the index is not in the middle of an escape
	index = ImproveExpandIndexEscape(rd, index, rp.res.Escape)

	index = rp.expandLeft(rd, index)
	if !rp.parse2(rd, index) {
		return rp.sc.Errorf("path")
	}
	return nil
}

func (rp *ResParser2) expandLeft(rd iorw.Reader, index int) int {
	rp.sc = scanutil.NewScanner(rd)
	rp.sc.SetStartPos(index)
	rp.sc.Reverse = true // to the left
	// worst case scenario left expansion
	_, _ = rp.integer() // column
	_ = rp.colon()
	_, _ = rp.integer() // line
	_ = rp.colon()
	_, _ = rp.path()
	_ = rp.prePath()
	return rp.sc.Pos
}

func (rp *ResParser2) parse2(rd iorw.Reader, index int) bool {
	rd2 := iorw.NewLimitedReader(rd, index, index+2000, 0)

	rp.sc = scanutil.NewScanner(rd2)
	rp.sc.SetStartPos(index)

	// pre path
	pp := rp.prePath()

	// path
	s, ok := rp.path()
	if !ok {
		return false
	}
	s = pp + s
	rp.res.RawPath = s
	rp.res.Path = RemoveFilenameEscapes(s, rp.res.Escape, rp.res.PathSep)

	// line/column
	if rp.colon() {
		v, ok := rp.integer()
		if ok {
			rp.res.Line = v
			if rp.colon() {
				v, ok := rp.integer()
				if ok {
					rp.res.Column = v
				}
			}
		}
	}

	rp.res.ExpandedMin = index
	rp.res.ExpandedMax = rp.sc.Pos

	return true
}

//----------

func (rp *ResParser2) colon() bool {
	if rp.sc.Match.Rune(':') {
		rp.sc.Advance()
		return true
	}
	return false
}

func (rp *ResParser2) path() (string, bool) {
	ok := rp.sc.RewindOnFalse(func() bool {
		for {
			if rp.sc.Match.Escape(rp.res.Escape) {
				continue
			}
			if rp.sc.Match.Fn(func(ru rune) bool {
				return ru == rp.res.PathSep ||
					unicode.IsLetter(ru) || unicode.IsDigit(ru) ||
					strings.ContainsRune(PathExtraRunes, ru)
			}) {
				continue
			}
			break
		}
		return !rp.sc.Empty()
	})
	if !ok {
		return "", false
	}
	s := rp.sc.Value()
	rp.sc.Advance()
	return s, true
}

func (rp *ResParser2) integer() (res int, ok bool) {
	_ = rp.sc.RewindOnFalse(func() bool {
		v, err := rp.sc.Match.IntValueAdvance()
		if err != nil {
			return false
		}
		res = v
		ok = true
		return true
	})
	return
}

//----------

func (rp *ResParser2) prePath() string {
	// ignore/moveforward "file://" prefix
	_, ok := rp.filePrefix()
	if ok {
		rp.res.ExpandedMin = rp.sc.Pos
		rp.res.PathSep = '/' // enforce this path seperator
	}
	return rp.volume()
}

func (rp *ResParser2) filePrefix() (string, bool) {
	ok := rp.sc.RewindOnFalse(func() bool {
		return rp.sc.Match.Sequence("file://") && rp.sc.PeekRune() == '/'
	})
	if !ok {
		return "", false
	}
	rp.res.ExpandedMin = rp.sc.Pos
	s := rp.sc.Value()
	rp.sc.Advance()
	return s, true
}

func (rp *ResParser2) volume() string {
	if !rp.res.ParseVolume {
		return ""
	}
	ok := rp.sc.Match.FnOrder(
		func() bool {
			if !rp.sc.Reverse {
				rp.sc.Reverse = true
				defer func() { rp.sc.Reverse = false }()
			}
			return !unicode.IsLetter(rp.sc.PeekRune())
		},
		func() bool {
			return rp.sc.Match.Fn(unicode.IsLetter)
		},
		func() bool {
			return rp.sc.Match.Rune(':')
		},
		func() bool {
			if rp.sc.Reverse {
				rp.sc.Reverse = false
				defer func() { rp.sc.Reverse = true }()
			}
			ru := rp.sc.PeekRune()
			//return ru == rp.res.PathSep
			return ru == '/' || ru == '\\' // peek either slash
		},
	)
	if !ok {
		return ""
	}
	s := rp.sc.Value()
	rp.sc.Advance()
	return s
}
