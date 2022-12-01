package reslocparser

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/jmigpin/editor/util/parseutil"
)

type ResLoc struct {
	Path   string // raw path
	Line   int    // 0 is nil
	Column int    // 0 is nil
	Offset int    // <=-1 is nil // TODO: file:#123?

	PathSep rune
	Escape  rune

	Scheme string // ex: "file://", useful to know when to translate to another path separator
	Volume string

	Pos, End int // contains reverse expansion
}

func (rl *ResLoc) ClearFilename1() string {
	s := rl.Path

	// commented: don't remove escapes to allow stringify to destinguish between other params (in case ":" was escaped)
	//s = parseutil.RemoveEscapes(s, rl.escape)

	s = parseutil.CleanMultiplePathSeps(s, rl.PathSep)

	if rl.Scheme == "file://" {
		// bypass first slash
		if rl.Volume != "" {
			rs := []rune(s)
			if rs[0] == '/' {
				s = string(rs[1:])
			}
		}
		// replace slashes
		sep := '/'
		if rl.PathSep != sep {
			s = strings.Replace(s, string(sep), string(rl.PathSep), -1)
		}
	}

	//if rl.separator2 != 0 {
	//}
	//u, err := strconv.Unquote("\"" + s + "\"")
	//if err == nil {
	//	s = u
	//}

	if u, err := url.QueryUnescape(s); err == nil {
		s = u
	}

	return s
}

func (rl *ResLoc) Stringify1() string {
	s := rl.ClearFilename1()
	if rl.Line > 0 {
		s += fmt.Sprintf(":%d", rl.Line)
	}
	if rl.Column > 0 {
		s += fmt.Sprintf(":%d", rl.Column)
	}
	return s
}
