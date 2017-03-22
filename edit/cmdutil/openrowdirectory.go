package cmdutil

import (
	"os"
	"path"
)

func OpenRowDirectory(erow ERower) {
	tsd := erow.ToolbarSD()
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

	ed := erow.Ed()
	erow2, ok := ed.FindERow(p)
	if !ok {
		col := erow.Row().Col
		i := col.RowIndex(erow.Row()) + 1
		erow2 = ed.NewERow(p, col, i)
		err = erow2.LoadContentClear()
		if err != nil {
			erow.Ed().Error(err)
			return
		}
	}
	erow2.Row().Square.WarpPointer()
}
