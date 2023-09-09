package godebug

import (
	"bytes"
	"fmt"
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
		ok2 := t.Run(name, func(t2 *testing.T) {
			if err := testAnnotator2(t2, name, file.Data, file2.Data, testsFilename, line, ar, iar+1); err != nil {
				t2.Fatal(err)
			}
		})
		if !ok2 {
			break // stop on first failed test
		}
	}
}
func testAnnotator2(t *testing.T, name string, in, out []byte, filename string, line int, ar *txtar.Archive, iarOut int) error {
	t.Logf("name: %v\n", name)

	fset := token.NewFileSet()

	// parse input ast
	mode := parser.ParseComments
	astFile, err := parser.ParseFile(fset, "a.go", in, mode)
	if err != nil {
		return err
	}

	// annotate
	ann := NewAnnotator(fset)
	ann.simpleTestMode = true
	ann.debugPkgName = "Σ"   // expected by tests
	ann.debugVarPrefix = "Σ" // expected by tests

	_, _, _ = testutil.CollectLog(t, func() error {
		// TODO: types and other? anntype?
		ann.AnnotateAstFile(astFile)
		return nil
	})

	// output result to string for comparison
	res := ann.sprintNode(astFile)
	//_ := parseutil.TrimLineSpaces(res) // old way of comparing

	fail := res != string(out)

	overwrite := false
	if !overwrite {
		// use after flag "--":
		// ex: go test -run=Annotator -- -overwriteoutput=TestAnnotator1.in
		v, ok := flagutil.GetFlagString(os.Args, "owout")
		overwrite = ok && v == name
	}
	if !overwrite {
		v := flagutil.GetFlagBool(os.Args, "owoutFirstFail")
		overwrite = v && fail
	}
	continueOnOverwrite := false
	if !overwrite {
		v := flagutil.GetFlagBool(os.Args, "owoutAllFail")
		overwrite = v && fail
		if overwrite {
			continueOnOverwrite = true
		}
	}
	if overwrite {
		fmt.Printf("overwriting output for test: %v\n", name)
		ar.Files[iarOut].Data = []byte(res)
		b := txtar.Format(ar)
		//fmt.Println(string(b)) // DEBUG
		if err := os.WriteFile(filename, b, 0o644); err != nil {
			return err
		}
		if continueOnOverwrite {
			return nil
		}
		return fmt.Errorf("tests file overwriten")
	}

	if fail {
		location := fmt.Sprintf("%s:%d", filename, line)
		err := fmt.Errorf(""+ //"\n"+
			"%s\n"+
			"-- input --\n%s"+ // has ending newline (go fmt)
			"-- result --\n%s"+ // has ending newline (go fmt)
			"-- expected --\n%s", // has ending newline (go fmt)
			location, in, res, out)
		return err
	}
	return nil
}
