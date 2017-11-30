package contentcmd

import (
	"github.com/jmigpin/editor/core/cmdutil"
	"github.com/jmigpin/editor/ui/tautil"
)

func directory(erow cmdutil.ERower) bool {
	ta := erow.Row().TextArea

	var str string
	if ta.SelectionOn() {
		a, b := tautil.SelectionStringIndexes(ta)
		str = ta.Str()[a:b]
	} else {
		str = expandLeftRightStopRunes(ta.Str(), ta.CursorIndex(), "\"'`=:<>()[]")
	}

	if str == "" {
		return false
	}

	dir, fi, ok := findFileinfo(erow, str)
	if !ok || !fi.IsDir() {
		return false
	}

	ed := erow.Ed()
	erow2, ok := ed.FindERower(dir)
	if !ok {
		col := erow.Row().Col
		u, ok := erow.Row().NextRow()
		if !ok {
			u = nil
		}
		erow2 = ed.NewERowerBeforeRow(dir, col, u)
		err := erow2.LoadContentClear()
		if err != nil {
			ed.Error(err)
		}
	}
	erow2.Row().Flash()
	return true
}
