package gosource

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/token"
	"path/filepath"
	"strings"
)

func FullFilename(filename string) string {
	if filepath.IsAbs(filename) {
		return filename
	}
	// pkg filename
	dir := filepath.Dir(filename)
	bpkg, _ := build.Import(dir, ".", build.FindOnly)
	if bpkg.Dir == "" {
		return filename
	}
	return filepath.Join(bpkg.Dir, filepath.Base(filename))
}

func FullDirectory(dir string) string {
	if filepath.IsAbs(dir) {
		return dir
	}
	// pkg dir
	bpkg, _ := build.Import(dir, ".", build.FindOnly)
	if bpkg.Dir == "" {
		return dir
	}
	return bpkg.Dir
}

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

func PkgName(path string) (string, error) {
	// transform into pkg dir
	pkgDir := path
	srcDir := "."
	if filepath.IsAbs(pkgDir) {
		srcDir, pkgDir = ExtractSrcDir(pkgDir)
	}
	// pkg dir
	bpkg, err := build.Import(pkgDir, srcDir, 0)
	if err != nil {
		return "", err
	}
	return bpkg.Name, nil
}

func ExtractSrcDir(filename string) (string, string) {
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

func PrintInspect(root ast.Node) {
	d := -1
	ast.Inspect(root, func(node ast.Node) bool {
		if node == nil {
			d--
			return false
		}
		d++
		fmt.Printf("%*s%T %v\n", d*4, "", node, node)
		return true
	})
}

func TokPositionStr(str string, pos token.Position) (string, error) {
	margin := 50
	min, max := pos.Offset-margin, pos.Offset+margin
	if min < 0 {
		min = 0
	}
	if max > len(str) {
		max = len(str)
	}

	s := str[min:max]

	o := pos.Offset - min
	ru := rune(9679) // centered dot
	s = s[:o] + string(ru) + s[o:]

	return s, nil
}
