package cmdutil

import (
	"log"
	"os"
	"path"

	"github.com/jmigpin/editor/ui"
)

func RowDirectory(ed Editorer, erow ERower) {
	fp := erow.Filename()
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

	log.Print("open row dir", p)

	col := erow.Row().Col
	next := erow.Row().NextRow()
	erows := directoryRows(ed, p, col, next)
	for _, e := range erows {
		e.Flash()
	}
}

func OpenDirectoryRow(ed Editorer, path string, col *ui.Column, next *ui.Row) {
	_ = directoryRows(ed, path, col, next)
}

func directoryRows(ed Editorer, path string, col *ui.Column, next *ui.Row) []ERower {
	erows := ed.FindERowers(path)
	if len(erows) > 0 {
		return erows
	}

	// new row
	erow := ed.NewERowerBeforeRow(path, col, next)
	err := erow.LoadContentClear()
	if err != nil {
		erow.Ed().Error(err)
	}
	return []ERower{erow}
}
