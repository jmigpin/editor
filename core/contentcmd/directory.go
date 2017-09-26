package contentcmd

import (
	"os"
	"path"

	"github.com/jmigpin/editor/core/cmdutil"
)

func directory(erow cmdutil.ERower, p string) bool {
	if p == "" {
		return false
	}
	if !path.IsAbs(p) {
		p = path.Join(erow.Dir(), p)
	}
	fi, err := os.Stat(p)
	if err != nil {
		return false
	}
	if !fi.IsDir() {
		return false
	}

	ed := erow.Ed()
	erow2, ok := ed.FindERow(p)
	if !ok {
		col := erow.Row().Col
		u, ok := erow.Row().NextSiblingRow()
		if !ok {
			u = nil
		}
		erow2 = ed.NewERowBeforeRow(p, col, u)
		err = erow2.LoadContentClear()
		if err != nil {
			ed.Error(err)
		}
	}
	erow2.Row().WarpPointer()
	return true
}
