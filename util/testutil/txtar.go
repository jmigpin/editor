package testutil

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/jmigpin/editor/util/pathutil"
	"golang.org/x/tools/txtar"
)

func ParseTxtar(src []byte, filename string) *Archive {
	tar := txtar.Parse(src)

	ar := &Archive{}
	ar.Filename = filename
	ar.Tar = tar

	line := countLines(tar.Comment)
	for _, f := range tar.Files {
		line++ // file header line
		ar.Lines = append(ar.Lines, line)
		line += countLines(f.Data)
	}
	return ar
}

//----------

type Archive struct {
	Tar      *txtar.Archive
	Filename string // for errors
	Lines    []int  // Tar.Files[] line position in src
}

func (ar *Archive) Error(err error, i int) error {
	return fmt.Errorf("%s:%d: %w", ar.Filename, ar.Lines[i]+1, err)
}

//----------
//----------
//----------

func RunArchive2(t *testing.T, ar *Archive,
	fn func(t2 *testing.T, name string, input, output []byte) error,
) {
	RunArchive(t, ar, []string{".in", ".out"},
		func(t2 *testing.T, name string, data [][]byte) error {
			return fn(t2, name, data[0], data[1])
		},
	)
}

// Expects n files named in filesExts args
func RunArchive(t *testing.T, ar *Archive, filesExts []string,
	fn func(t2 *testing.T, name string, datas [][]byte) error,
) {
	// map files
	fm := map[string]txtar.File{}
	for _, file := range ar.Tar.Files {
		if _, ok := fm[file.Name]; ok {
			t.Fatalf("file already defined: %v", file.Name)
		}
		fm[file.Name] = file
	}

	for fi, file := range ar.Tar.Files {
		// get data
		datas := [][]byte{}
		for i := 0; i < len(filesExts); i++ {
			ext := filesExts[i]
			fname := pathutil.ReplaceExt(file.Name, ext)

			// run only files that match the first ext
			if i == 0 {
				if fname != file.Name {
					break
				}
			}

			f, ok := fm[fname]
			if !ok {
				// show warning only for files after first
				if i > 0 {
					t.Logf("warning: missing %q for %v", ext, file.Name)
				}
				break
			}
			datas = append(datas, f.Data)
		}
		if len(datas) != len(filesExts) {
			continue
		}

		name := filepath.Base(file.Name)
		ok2 := t.Run(name, func(t2 *testing.T) {
			t2.Logf("run name: %v", name)
			err := fn(t2, name, datas)
			if err != nil {
				t2.Fatal(ar.Error(err, fi))
			}
		})
		if !ok2 {
			break // stop on first failed test
		}
	}
}

//----------
//----------
//----------

func countLines(b []byte) int {
	return bytes.Count(b, []byte("\n"))
}
