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
		isStop := StopOnSpaceAndRunesFn(FilenameStopRunes)
		l, r := expandLeftRightStop(ta.Str(), ta.CursorIndex(), isStop)
		str = ta.Str()[l:r]
	}

	if str == "" {
		return false
	}

	dir, fi, ok := findFileinfo(erow, str)
	if !ok || !fi.IsDir() {
		return false
	}

	ed := erow.Ed()
	erows := ed.FindERowers(dir)
	if len(erows) == 0 {
		col := erow.Row().Col
		next := erow.Row().NextRow()
		erow2 := ed.NewERowerBeforeRow(dir, col, next)
		err := erow2.LoadContentClear()
		if err != nil {
			ed.Error(err)
		}
		erows = []cmdutil.ERower{erow2}
	}
	for _, e := range erows {
		e.Flash()
	}
	return true
}
