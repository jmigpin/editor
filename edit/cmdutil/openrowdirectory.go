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

	ed := erow.Editorer()
	col := ed.ActiveColumn()
	erow2 := ed.FindERowOrCreate(p, col)
	err = erow2.LoadContentClear()
	if err == nil {
		erow2.Row().Square.WarpPointer()
	}
}
