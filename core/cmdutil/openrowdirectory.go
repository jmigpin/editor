package cmdutil

import (
	"os"
	"path"

	"github.com/jmigpin/editor/ui"
)

func OpenRowDirectory(erow ERower) {
	ed := erow.Ed()

	fp := erow.DecodedPart0Arg0()
	p := path.Dir(fp) // if fp=="", dir returns "."

	fi, err := os.Stat(p)
	if err != nil {
		ed.Error(err)
		return
	}
	if !fi.IsDir() {
		ed.Errorf("not a directory: %v", p)
		return
	}

	next, _ := erow.Row().NextSiblingRow()
	OpenDirectoryRow(ed, p, erow.Row().Col, next)
}
func OpenDirectoryRow(ed Editorer, path string, col *ui.Column, next *ui.Row) {
	erow, ok := ed.FindERow(path)
	if !ok {
		erow = ed.NewERowBeforeRow(path, col, next)
		err := erow.LoadContentClear()
		if err != nil {
			erow.Ed().Error(err)
			return
		}
	}
	erow.Row().WarpPointer()
}
