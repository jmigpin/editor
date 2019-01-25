package goutil

import (
	"go/build"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmigpin/editor/util/osutil"
)

func GoPath() []string {
	// TODO: use go/build defaultgopath if it becomes public

	a := []string{}

	add := func(b ...string) { a = append(a, b...) }

	gopath := os.Getenv("GOPATH")
	if gopath != "" {
		add(filepath.SplitList(gopath)...)
	} else {
		// from go/build/build.go:270:3
		add(filepath.Join(osutil.HomeEnvVar(), osutil.GoExec()))
	}

	return a
}

//----------

func ExtractSrcDir(filename string) (string, string) {
	// TODO: can't do this here since abs will use current dir
	//u, err := filepath.Abs(filename)
	//if err == nil {
	//	filename = u
	//}

	srcDir := ""
	for _, d := range build.Default.SrcDirs() {
		d += "/"
		if strings.HasPrefix(filename, d) {
			srcDir = filename[:len(d)]
			filename = filename[len(d):]
			return srcDir, filename
		}
	}
	return srcDir, filename
}

//----------

func PkgFilenames(dir string, testFiles bool) (string, string, []string, error) {
	// transform into pkg dir
	pkgDir := dir
	srcDir := "."
	if filepath.IsAbs(dir) {
		srcDir, pkgDir = ExtractSrcDir(dir)
	}
	// pkg dir
	bpkg, err := build.Import(pkgDir, srcDir, 0)
	if err != nil {
		return dir, pkgDir, nil, err
	}
	a := append(bpkg.GoFiles, bpkg.CgoFiles...)
	if testFiles {
		a = append(a, bpkg.TestGoFiles...)
	}
	return bpkg.Dir, pkgDir, a, nil
}
