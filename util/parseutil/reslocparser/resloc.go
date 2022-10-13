package reslocparser

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/jmigpin/editor/util/parseutil"
	"github.com/jmigpin/editor/util/parseutil/lrparser"
)

type ResLoc struct {
	Scheme   string // "file://"
	Filename string
	Line     int // -1 if not set
	Col      int // -1 if not set
	Offset   int // -1 if not set

	Bnd *lrparser.BuildNodeData // used in testing // TODO: clear if not testing

	// TODO: expanded min/max

	escape     rune
	separator  rune
	separator2 rune // windows: translating file://c:/a/b to c:\a\b
}

func (rl *ResLoc) ClearFilename1() string {
	s := rl.Filename

	// commented: don't remove escapes to allow stringify to destinguish between other params (in case ":" was escaped)
	//s = parseutil.RemoveEscapes(s, rl.escape)

	s = parseutil.CleanMultiplePathSeps(s, rl.separator)

	if rl.separator2 != 0 {
		s = strings.Replace(s, string(rl.separator2), string(rl.separator), -1)
	}
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
	if rl.Col > 0 {
		s += fmt.Sprintf(":%d", rl.Col)
	}
	return s
}
