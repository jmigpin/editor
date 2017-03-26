package cmdutil

import (
	"os"
	"path"
)

func OpenRowDirectory(erow ERower) {
	ed := erow.Ed()

	p := ""

	tsd := erow.ToolbarSD()
	fp, ok := tsd.DecodePart0Arg0()
	if ok {
		p = path.Dir(fp)
	}

	fi, err := os.Stat(p)
	if err != nil {
		ed.Error(err)
		return
	}
	if !fi.IsDir() {
		ed.Errorf("not a directory: %v", p)
		return
	}

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
	erow2.Row().WarpPointer()
}
