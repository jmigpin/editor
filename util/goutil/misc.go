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
		// from go/build/build.go:274
		add(filepath.Join(osutil.HomeEnvVar(), "go"))
	}
	return a
}

//----------

func ExtractSrcDir(filename string) (string, string) {
	srcDir := ""
	for _, d := range build.Default.SrcDirs() {
		d += string(filepath.Separator)
		if strings.HasPrefix(filename, d) {
			srcDir = filename[:len(d)]
			filename = filename[len(d):]
			return srcDir, filename
		}
	}
	return srcDir, filename
}
