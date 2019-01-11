package gosource

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/token"
	"path/filepath"
	"unicode"
	"unicode/utf8"

	"github.com/jmigpin/editor/util/goutil"
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
		srcDir, pkgDir = goutil.ExtractSrcDir(dir)
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
		srcDir, pkgDir = goutil.ExtractSrcDir(pkgDir)
	}
	// pkg dir
	bpkg, err := build.Import(pkgDir, srcDir, 0)
	if err != nil {
		return "", err
	}
	return bpkg.Name, nil
}

//func ExtractSrcDir(filename string) (string, string) {
//	u, err := filepath.Abs(filename)
//	if err == nil {
//		filename = u
//	}

//	srcDir := ""
//	for _, d := range build.Default.SrcDirs() {
//		d += "/"
//		if strings.HasPrefix(filename, d) {
//			srcDir = filename[:len(d)]
//			filename = filename[len(d):]
//			return srcDir, filename
//		}
//	}
//	return srcDir, filename
//}

//------------

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

//------------

// Prints a special rune at the position in the string. Used for debugging.
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

//------------

//// Allows a src string to have multiple cursor strings to simulate cursor position. Used in testing.
//func SourceCursor(cursorStr, src string, n int) (string, int, error) {
//	// cursor positions
//	pos := []int{}
//	k := 0
//	for {
//		j := strings.Index(src[k:], cursorStr)
//		if j < 0 {
//			break
//		}
//		k += j
//		pos = append(pos, k)
//		k++
//	}

//	// nth position
//	if n >= len(pos) {
//		return "", 0, fmt.Errorf("nth index not found: n=%v, len=%v", n, len(pos))
//	}
//	index := pos[n]

//	// remove cursors
//	index -= n * len(cursorStr)
//	src2 := strings.Replace(src, cursorStr, "", -1)

//	return src2, index, nil
//}

//------------

func InsertSemicolon(str string, index int) string {
	return str[:index] + ";" + str[index:]
}

func InsertSemicolonAfterIdent(str string, index int) string {
	j := index
	for i, ru := range str[index:] {
		j = index + i
		if !unicode.IsLetter(ru) {
			break
		}
	}
	return InsertSemicolon(str, j)
}

func BackTrackSpaceInsertSemicolon(str string, index int) (int, string) {
	j := index
	for j > 0 {
		ru, size := utf8.DecodeLastRuneInString(str[:j])
		if ru != ' ' {
			break
		}
		j -= size
	}
	return index - j, InsertSemicolon(str, j)
}

//------------

//// taken from: go/parser/interface.go:25:9
//func ReadSource(filename string, src interface{}) ([]byte, error) {
//	if src != nil {
//		switch s := src.(type) {
//		case string:
//			return []byte(s), nil
//		case []byte:
//			return s, nil
//		case *bytes.Buffer:
//			// is io.Reader, but src is already available in []byte form
//			if s != nil {
//				return s.Bytes(), nil
//			}
//		case io.Reader:
//			var buf bytes.Buffer
//			if _, err := io.Copy(&buf, s); err != nil {
//				return nil, err
//			}
//			return buf.Bytes(), nil
//		}
//		return nil, errors.New("invalid source")
//	}
//	return ioutil.ReadFile(filename)
//}
