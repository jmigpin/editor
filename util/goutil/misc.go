package goutil

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmigpin/editor/util/osutil"
)

func GoPath() []string {
	// TODO: use go/build defaultgopath if it becomes public
	a := []string{}
	gopath := os.Getenv("GOPATH")
	if gopath != "" {
		a = append(a, filepath.SplitList(gopath)...)
	} else {
		// from go/build/build.go:274
		a = append(a, filepath.Join(osutil.HomeEnvVar(), "go"))
	}
	return a
}

func JoinPathLists(w ...string) string {
	return strings.Join(w, string(os.PathListSeparator))
}

//----------

//func ExtractSrcDir(filename string) (string, string) {
//	srcDir := ""
//	for _, d := range build.Default.SrcDirs() {
//		d += string(filepath.Separator)
//		if strings.HasPrefix(filename, d) {
//			srcDir = filename[:len(d)]
//			filename = filename[len(d):]
//			return srcDir, filename
//		}
//	}
//	return srcDir, filename
//}

//----------

func AstFileFilename(astFile *ast.File, fset *token.FileSet) (string, error) {
	if astFile == nil {
		panic("!")
	}
	tfile := fset.File(astFile.Package)
	if tfile == nil {
		return "", fmt.Errorf("not found")
	}
	return tfile.Name(), nil
}
