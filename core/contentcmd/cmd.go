package contentcmd

import (
	"os"
	"path"
	"strings"
	"unicode"

	"github.com/jmigpin/editor/core/cmdutil"
)

func Cmd(erow cmdutil.ERower) {
	if ok := goSource(erow); ok {
		return
	}
	if ok := filePos(erow); ok {
		return
	}
	if ok := directory(erow); ok {
		return
	}
	if ok := openSession(erow); ok {
		return
	}
	if ok := http(erow); ok {
		return
	}
	erow.Ed().Errorf("no content cmd was successful")
}

func expandLeftRightStopRunes(str string, index int, stopRunes string) string {
	isStop := func(ru rune) bool {
		if unicode.IsSpace(ru) {
			return true
		}
		i := strings.IndexAny(string(ru), stopRunes)
		return i >= 0
	}
	return expandLeftRightStop(str, index, isStop)
}

func expandLeftRightStop(str string, index int, isStop func(rune) bool) string {
	i0 := strings.LastIndexFunc(str[:index], isStop)
	if i0 < 0 {
		i0 = 0
	} else {
		i0 += 1 // size of stop rune // TODO: rune size
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

// TODO: use expand left right
// Used to get argument after a command (ex: opensession)
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
	{
		u := path.Join(erow.Dir(), p)
		fi, err := os.Stat(u)
		if err == nil {
			return u, fi, true
		}
	}

	// go paths
	{
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
	}

	// c include paths
	{
		a := []string{
			"/usr/include",
		}
		for _, d := range a {
			u := path.Join(d, p)
			fi, err := os.Stat(u)
			if err == nil {
				return u, fi, true
			}
		}
	}

	return "", nil, false
}
