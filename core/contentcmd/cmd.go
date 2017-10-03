package contentcmd

import (
	"os"
	"path"
	"strings"
	"unicode"

	"github.com/jmigpin/editor/core/cmdutil"
	"github.com/jmigpin/editor/ui/tautil"
)

func Cmd(erow cmdutil.ERower) {
	var s string
	ta := erow.Row().TextArea

	if ta.SelectionOn() {
		a, b := tautil.SelectionStringIndexes(ta)
		s = ta.Str()[a:b]
	} else {
		s = expandLeftRight(ta.Str(), ta.CursorIndex())
	}

	if ok := file(erow, s); ok {
		return
	}
	if ok := directory(erow, s); ok {
		return
	}
	if ok := openSession(erow, s); ok {
		return
	}
	if ok := http(erow, s); ok {
		return
	}

	erow.Ed().Errorf("no content cmd was run")
}

func expandLeftRight(str string, index int) string {
	isStop := func(ru rune) bool {
		if unicode.IsSpace(ru) {
			return true
		}
		switch ru {
		case '"', '<', '>':
			return true
		}
		return false
	}

	i0 := strings.LastIndexFunc(str[:index], isStop)
	if i0 < 0 {
		i0 = 0
	} else {
		i0 += 1 // size of stop rune (quote or space)
	}
	i1 := strings.IndexFunc(str[index:], isStop)
	if i1 < 0 {
		i1 = len(str)
	} else {
		i1 += index
	}
	s2 := str[i0:i1]
	return s2
}

// Used on opensession command to get argument.
func afterSpaceExpandRightUntilSpace(str string, index int) string {
	if index > len(str) {
		index = len(str)
	}
	// find space
	i0 := strings.IndexFunc(str[index:], unicode.IsSpace)
	if i0 < 0 {
		return ""
	}
	i0 += index
	// pass all spaces
	isNotSpace := func(ru rune) bool { return !unicode.IsSpace(ru) }
	i2 := strings.IndexFunc(str[i0:], isNotSpace)
	if i2 < 0 {
		return ""
	}
	i2 += i0
	// find space
	i3 := strings.IndexFunc(str[i2:], unicode.IsSpace)
	if i3 < 0 {
		i3 = len(str)
	} else {
		i3 += i2
	}
	s2 := str[i2:i3]
	s3 := strings.TrimSpace(s2)
	return s3
}

// Used by "file" and "directory".
// Also checks in GOPATH and GOROOT.
func findFileinfo(erow cmdutil.ERower, p string) (string, os.FileInfo, bool) {
	// absolute path
	if path.IsAbs(p) {
		fi, err := os.Stat(p)
		if err == nil {
			return p, fi, true
		}
		return "", nil, false
	}

	// erow path
	u := path.Join(erow.Dir(), p)
	fi, err := os.Stat(u)
	if err == nil {
		return u, fi, true
	}

	// go paths
	gopath := os.Getenv("GOPATH")
	a := strings.Split(gopath, ":")
	a = append(a, os.Getenv("GOROOT"))
	for _, d := range a {
		u := path.Join(d, "src", p)
		fi, err := os.Stat(u)
		if err == nil {
			return u, fi, true
		}
	}

	return "", nil, false
}
