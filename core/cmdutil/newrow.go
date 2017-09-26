package cmdutil

import (
	"os"
	"path"
)

func NewRow(ed Editorer) {
	p := "."

	erow2, ok := ed.ActiveERow()
	if ok {
		fp := erow2.Filename()

		// stick with directory if exists, otherwise get base dir
		fi, err := os.Stat(fp)
		if err == nil && fi.IsDir() {
			p = fp
		} else {
			p = path.Dir(fp) // if fp=="", dir returns "."
		}
	}

	col, nextRow := ed.GoodColumnRowPlace()
	erow := ed.NewERowBeforeRow(p+" | ", col, nextRow)
	erow.Row().WarpPointer()
}
