package contentcmd

import (
	"go/token"
	"path"

	"github.com/jmigpin/editor/core/cmdutil"
	"github.com/jmigpin/editor/core/gosource"
)

func goSource(erow cmdutil.ERower, strIndex int) bool {
	if !erow.IsRegular() {
		return false
	}
	if path.Ext(erow.Filename()) != ".go" {
		return false
	}
	ta := erow.Row().TextArea
	pos, end, err := gosource.DeclPosition(erow.Filename(), ta.Str(), strIndex)
	if err != nil {
		//log.Print(err)
		return false
	}

	goSourceOpenPosition(erow, pos, end)

	return true
}

func goSourceOpenPosition(erow cmdutil.ERower, pos, end *token.Position) {
	ed := erow.Ed()
	m := make(map[cmdutil.ERower]bool)

	// any duplicate row that has the index already visible
	erows := ed.FindERowers(pos.Filename)
	for _, e := range erows {
		if e.Row().TextArea.IndexIsVisible(pos.Offset) {
			m[e] = true
		}
	}

	//// choose a duplicate row that is not the current row (old way).
	//// choose a duplicate row that is after the current row. Allows to maintain previous rows editing position while creating rows downward.
	//if len(m) == 0 {
	//	erows := ed.FindERowers(pos.Filename)
	//	ei := -1
	//	for i, e := range erows {
	//		if e == erow {
	//			ei = i
	//		}
	//		if e != erow && ei >= 0 && i > ei {
	//			m[e] = true
	//			break
	//		}
	//	}
	//}

	// open new row
	if len(m) == 0 {
		// place new row after erow
		col := erow.Row().Col
		next := erow.Row().NextRow()
		e := ed.NewERowerBeforeRow(pos.Filename, col, next)
		err := e.LoadContentClear()
		if err != nil {
			ed.Error(err)
			return
		}
		m[e] = true
	}

	// show position on selected rows
	for e, _ := range m {
		row2 := e.Row()
		row2.ResizeTextAreaIfVerySmall()
		ta2 := row2.TextArea
		ta2.MakeIndexVisible(pos.Offset)
		row2.TextArea.FlashIndexLen(pos.Offset, end.Offset-pos.Offset)
	}
}
