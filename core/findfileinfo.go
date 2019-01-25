package core

import (
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/jmigpin/editor/util/goutil"
)

// Checks in GOROOT/GOPATH,  and some C include dirs.
func FindFileInfo(name, dir string) (string, os.FileInfo, bool) {
	// absolute path
	if path.IsAbs(name) {
		fi, err := os.Stat(name)
		if err == nil {
			return name, fi, true
		}
		return "", nil, false
	}

	// join with dir
	{
		u := filepath.Join(dir, name)
		fi, err := os.Stat(u)
		if err == nil {
			return u, fi, true
		}
	}

	// go paths
	{
		a := []string{runtime.GOROOT()}
		a = append(a, goutil.GoPath()...)
		for _, d := range a {
			u := filepath.Join(d, "src", name)
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
			u := filepath.Join(d, name)
			fi, err := os.Stat(u)
			if err == nil {
				return u, fi, true
			}
		}
	}

	return "", nil, false
}
