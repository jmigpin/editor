package reslocparser

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/jmigpin/editor/util/parseutil"
)

type ResLoc struct {
	Path string // raw path

	Line   int // 0 is nil
	Column int // 0 is nil
	Offset int // -1 is nil

	PathSep rune
	Escape  rune

	Scheme string // ex: "file://", useful to know when to translate to another path separator
	Volume string

	Pos, End int // contains reverse expansion
}

func NewResLoc() *ResLoc {
	return &ResLoc{Offset: -1}
}

//----------

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
	return rl.ToLinecolString()
}

//----------

func (rl *ResLoc) ToLinecolString() string {
	s := rl.ClearFilename1()
	if rl.Line > 0 {
		s += rl.linecolToString()
	}
	return s
}
func (rl *ResLoc) ToOffsetString() string {
	s := rl.ClearFilename1()
	if rl.Offset >= 0 {
		s += rl.offsetToString()
	}
	return s
}

//----------

func (rl *ResLoc) offsetToString() string {
	return fmt.Sprintf(":o=%d", rl.Offset)
}
func (rl *ResLoc) linecolToString() string {
	s := fmt.Sprintf(":%d", rl.Line)
	if rl.Column > 0 {
		s += fmt.Sprintf(":%d", rl.Column)
	}
	return s
}

//----------

//func (rl *ResLoc) ToString() string {
//	return rl.ToString2(false)
//}
//func (rl *ResLoc) ToString2(preferOffset bool) string {
//	s := rl.ClearFilename1()
//	u := ""
//	if preferOffset && rl.Offset >= 0 {
//		u = rl.offsetToString()
//	} else if rl.Line > 0 {
//		u = rl.linecolToString()
//	} else if rl.Offset >= 0 {
//		u = rl.offsetToString()
//	}
//	if u != "" {
//		u = ":" + u
//	}
//	return s + u
//}
