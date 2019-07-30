package godebug

import (
	"context"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func TestFiles01(t *testing.T) {
	src := `
		package main
		import "github.com/jmigpin/editor/core/godebug/debug"
	`
	files := NewFiles(token.NewFileSet())
	doFilesSrc(t, files, src, false)
}

func TestFiles02(t *testing.T) {
	files := NewFiles(token.NewFileSet())
	doFiles(t, files, "", true)
}

func TestFiles03(t *testing.T) {
	filename := "../../editor.go"
	files := NewFiles(token.NewFileSet())
	doFiles(t, files, filename, false)
}

//----------

func doFilesSrc(t *testing.T, files *Files, src string, tests bool) {
	filename := "main.go"
	if tests {
		filename = "main_test.go"
	}
	tmpFile, tmpDir := createTmpFileFromSrc(t, filename, src)
	defer os.RemoveAll(tmpDir)
	doFiles(t, files, tmpFile, tests)
}

func doFiles(t *testing.T, files *Files, filename string, tests bool) {
	ctx := context.Background()
	u, err := files.Do(ctx, &filename, tests)
	if err != nil {
		t.Fatal(err)
	}
	//if len(u) == 0 {
	//t.Fatal()
	//}
	for _, k := range u {
		base := filepath.Base(k.Filename)
		t.Logf("%v %v %v\n", base, k.Type, k.DebugSrc)
	}
}

//----------
