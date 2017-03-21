package contentcmd

import (
	"os"
	"path"

	"github.com/jmigpin/editor/edit/cmdutil"
)

func directory(erow cmdutil.ERower, p string) bool {
	if p == "" {
		return false
	}
	if !path.IsAbs(p) {
		filepath, fi, ok := erow.FileInfo()
		if ok {
			if fi.IsDir() {
				p = path.Join(filepath, p)
			} else {
				p = path.Join(path.Dir(filepath), p)
			}
		}
	}
	fi, err := os.Stat(p)
	if err != nil {
		return false
	}
	if !fi.IsDir() {
		return false
	}
	ed := erow.Editorer()
	col := ed.ActiveColumn()
	erow2 := ed.FindERowOrCreate(p, col)
	err = erow2.LoadContentClear()
	if err == nil {
		erow2.Row().Square.WarpPointer()
	}
	return true
}
