package cmdutil

import (
	"os"
	"path"

	"github.com/jmigpin/editor/ui"
)

func OpenRowDirectory(ed Editorer, row *ui.Row) {
	tsd := ed.RowToolbarStringData(row)
	f, ok := tsd.FirstPartFilename()
	if !ok {
		return
	}
	p := path.Dir(f)

	fi, err := os.Stat(p)
	if err != nil {
		return
	}
	if !fi.IsDir() {
		return
	}

	col := ed.ActiveColumn()
	row, err = ed.FindRowOrCreateInColFromFilepath(p, col)
	if err == nil {
		row.Square.WarpPointer()
	}
}
