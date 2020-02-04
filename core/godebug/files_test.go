package godebug

import (
	"fmt"
	"go/token"
	"testing"
)

//----------

func newFilesFromSrcs(t *testing.T, srcs ...string) (*Files, []string) {
	t.Helper()
	files, names, err := newFilesFromSrcs2(srcs...)
	if err != nil {
		t.Fatal(err)
	}
	return files, names
}

// setup workable files without calling "files.do()" such that the program is not loaded but the commented nodes in the src can be tested by using the files.NodeAnnType function.
func newFilesFromSrcs2(srcs ...string) (*Files, []string, error) {
	fset := token.NewFileSet()
	files := NewFiles(fset, false)
	names := []string{}
	for i, src := range srcs {
		filename := fmt.Sprintf("test/src%v.go", i)
		names = append(names, filename)
		astFile, err := files.fullAstFile2(filename, []byte(src))
		if err != nil {
			return nil, nil, err
		}
		// setup files to use comments handling func
		if err := files.addCommentedFile2(filename, astFile); err != nil {
			return nil, nil, err
		}
		// mark as annotated to have file hash computed
		files.annFilenames[filename] = struct{}{}
	}
	// allow to later run annotatorset annotatefile (needs files hashes)
	files.doAnnFilesHashes()
	return files, names, nil
}
