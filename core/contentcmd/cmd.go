package contentcmd

import (
	"os"
	"path"
	"strings"
	"unicode"

	"github.com/jmigpin/editor/core/cmdutil"
)

func Cmd(erow cmdutil.ERower, index int) {
	if ok := goSource(erow, index); ok {
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

func expandLeftRightStop(str string, index int, isStop func(rune) bool) (int, int) {
	l := expandLeftStop(str, index, isStop)
	r := expandRightStop(str, index, isStop)
	return l, r
}
func expandLeftStop(str string, index int, isStop func(rune) bool) int {
	i := strings.LastIndexFunc(str[:index], isStop)
	if i < 0 {
		i = 0
	} else {
		i += 1 // size of stop rune // TODO: rune size
	}
	return i
}
func expandRightStop(str string, index int, isStop func(rune) bool) int {
	i := strings.IndexFunc(str[index:], isStop)
	if i < 0 {
		i = len(str)
	} else {
		i += index
	}
	return i
}
func NotStop(fn func(rune) bool) func(rune) bool {
	return func(ru rune) bool {
		return !fn(ru)
	}
}
func StopOnSpaceAndRunesFn(stopRunes string) func(rune) bool {
	return func(ru rune) bool {
		if unicode.IsSpace(ru) {
			return true
		}
		i := strings.IndexAny(string(ru), stopRunes)
		return i >= 0
	}
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
