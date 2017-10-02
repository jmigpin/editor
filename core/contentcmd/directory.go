package contentcmd

import "github.com/jmigpin/editor/core/cmdutil"

func directory(erow cmdutil.ERower, p string) bool {
	if p == "" {
		return false
	}
	dir, fi, ok := findFileinfo(erow, p)
	if !ok || !fi.IsDir() {
		return false
	}

	ed := erow.Ed()
	erow2, ok := ed.FindERow(dir)
	if !ok {
		col := erow.Row().Col
		u, ok := erow.Row().NextSiblingRow()
		if !ok {
			u = nil
		}
		erow2 = ed.NewERowBeforeRow(dir, col, u)
		err := erow2.LoadContentClear()
		if err != nil {
			ed.Error(err)
		}
	}
	erow2.Row().WarpPointer()
	return true
}
