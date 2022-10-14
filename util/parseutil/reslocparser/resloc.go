package reslocparser

import (
	"fmt"
	"net/url"

	"github.com/jmigpin/editor/util/parseutil"
	"github.com/jmigpin/editor/util/parseutil/lrparser"
)

type ResLoc struct {
	Path   string // raw path
	Line   int    // 0 is nil
	Column int    // 0 is nil

	Scheme string // "file://"
	Offset int    // <=-1 is nil // TODO

	Pos, End int // contains reverse expansion

	Escape  rune
	PathSep rune
	//separator2 rune // windows: translating file://c:/a/b to c:\a\b

	Bnd *lrparser.BuildNodeData // used in testing // TODO: clear if not testing
}

func (rl *ResLoc) ClearFilename1() string {
	s := rl.Path

	// commented: don't remove escapes to allow stringify to destinguish between other params (in case ":" was escaped)
	//s = parseutil.RemoveEscapes(s, rl.escape)

	s = parseutil.CleanMultiplePathSeps(s, rl.PathSep)

	//if rl.separator2 != 0 {
	//	s = strings.Replace(s, string(rl.separator2), string(rl.PathSep), -1)
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
