package godebug

import (
	"fmt"
	"go/token"
	"testing"
)

//----------

func newFilesFromSrcs(t *testing.T, srcs ...string) *Files {
	t.Helper()
	files, err := newFilesFromSrcs2(srcs...)
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// setup workable files without calling "files.do()" such that the program is not loaded but the commented nodes in the src can be tested by using the files.NodeAnnType function.
func newFilesFromSrcs2(srcs ...string) (*Files, error) {
	fset := token.NewFileSet()
	files := NewFiles(fset, "", false, false, nil)
	for i, src := range srcs {
		filename := fmt.Sprintf("test/src%v.go", i)
		astFile, err := files.fullAstFile2(filename, []byte(src))
		if err != nil {
			return nil, err
		}
		// mark as annotated to have file hash computed
		f := files.NewFile(filename, FTSrc, nil)
		f.action = FAAnnotate
		// setup files to use comments handling func
		if err := files.findCommentedFile2(f, astFile); err != nil {
			return nil, err
		}
	}
	// allow to later run annotatorset annotatefile (needs files hashes)
	files.doAnnFilesHashes()
	return files, nil
}
