package godebug

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmigpin/editor/util/flagutil"
	"github.com/jmigpin/editor/util/pathutil"
	"github.com/jmigpin/editor/util/testutil"
	"golang.org/x/tools/txtar"
)

func TestAnnotator(t *testing.T) {
	testsFilename := "testdata/annotator/annotator_in_out.txt"
	ar, err := txtar.ParseFile(testsFilename)
	if err != nil {
		t.Fatal()
	}

	// map files
	m := map[string]txtar.File{}
	for _, file := range ar.Files {
		m[file.Name] = file
	}

	countLines := func(b []byte) int {
		return bytes.Count(b, []byte("\n"))
	}

	line := countLines(ar.Comment) + 1 // start at line 1
	nextLines := 0
	contentChanged := false
	for iar, file := range ar.Files {
		line += nextLines
		nextLines = countLines(file.Data) + 1 // add name line

		// only start a test with a ".in" ext
		if !strings.HasSuffix(file.Name, ".in") {
			continue
		}
		// find ".out"
		outName := pathutil.ReplaceExt(file.Name, ".out")
		file2, ok := m[outName]
		if !ok {
			t.Logf("warning: missing *.out for %v", file.Name)
			continue
		}

		name := filepath.Base(file.Name)
		stop := false
		ok2 := t.Run(name, func(t2 *testing.T) {
			stop2, changed, err := testAnnotator2(t2, name, file.Data, file2.Data, testsFilename, line, ar, iar+1)
			if err != nil {
				t2.Fatal(err)
			}
			if changed {
				contentChanged = true
			}
			if stop2 {
				stop = true
			}
		})
		if !ok2 || stop {
			break
		}
	}

	if contentChanged {
		b := txtar.Format(ar)
		//fmt.Println(string(b)) // DEBUG
		if err := os.WriteFile(testsFilename, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
func testAnnotator2(t *testing.T, name string, in0, out []byte, filename string, line int, ar *txtar.Archive, iarOut int) (bool, bool, error) {
	//t.Logf("name: %v\n", name)
	location := fmt.Sprintf("%s:%d", filename, line)

	// simplify input: add package line if not present
	in := in0
	if !bytes.HasPrefix(in, []byte("package ")) {
		in = append([]byte("package p1\n\n"), in...)
	}

	fset := token.NewFileSet()

	// parse input ast
	mode := parser.ParseComments
	astFile, err := parser.ParseFile(fset, "a.go", in, mode)
	if err != nil {
		return false, false, err
	}

	ti, err := getTypesInfo(fset, astFile)
	if err != nil {
		return false, false, fmt.Errorf("%v: %v", location, err)
	}

	// annotate
	ann := NewAnnotator(fset, ti)

	_, _, _ = testutil.CollectLog(t, func() error {
		ann.AnnotateAstFile(astFile)
		return nil
	})

	// find node to output (simplify output)
	node := (ast.Node)(astFile)
	ast.Inspect(astFile, func(n ast.Node) bool {
		if node == astFile {
			if _, ok := n.(*ast.FuncDecl); ok {
				node = n
			}
			return true
		}
		return false
	})

	// output result to string for comparison
	res := ann.sprintNode(node)
	if len(res) > 0 && res[len(res)-1] != '\n' {
		res = res + "\n"
	}

	//_ := parseutil.TrimLineSpaces(res) // old way of comparing

	fail := res != string(out)

	//----------

	// use after flag "--":
	// ex: go test -run=Annotator/TestAnnotator1.in -- -owout
	if fail {
		owOut := flagutil.GetFlagBool(os.Args, "owout")
		if owOut {
			fmt.Printf("overwrite test output: %v\n", name)
			ar.Files[iarOut].Data = []byte(res)

			owAll := flagutil.GetFlagBool(os.Args, "owall")
			return !owAll, owOut, nil
		}
	}

	if fail {
		err := fmt.Errorf(""+ //"\n"+
			"%s\n"+
			"-- input --\n%s"+ // has ending newline (go fmt)
			"-- result --\n%s"+ // has ending newline (go fmt)
			"-- expected --\n%s", // has ending newline (go fmt)
			location, in0, res, out)
		return false, false, err
	}

	return false, false, nil
}
