package core

import (
	"github.com/jmigpin/editor/core/parseutil"
	"github.com/jmigpin/editor/ui"
)

//----------

func OpenERowFilePosVisibleOrNew(ed *Editor, fp *parseutil.FilePos, rowPos *ui.RowPos) {
	// missing line/col
	if fp.Line == 0 {
		openERowFileOrNew(ed, fp, rowPos)
		return
	}

	info := ed.ReadERowInfo(fp.Filename)

	// rows to flash
	flash := map[*ERow]bool{}
	setindex := map[*ERow]bool{}

	// read file
	var str string
	if len(info.ERows) > 0 {
		str = info.ERows[0].Row.TextArea.Str()
	} else {
		erow, err := info.NewERow(rowPos)
		if err != nil {
			ed.Error(err)
			return
		}
		str = erow.Row.TextArea.Str()
		flash[erow] = true
		setindex[erow] = true
	}

	// offset index
	index := parseutil.LineColumnIndex(str, fp.Line, fp.Column)

	// add rows that have the index already visible to be flashed as well
	for _, e := range info.ERows {
		if e.Row.TextArea.IsIndexVisible(index) {
			flash[e] = true
		}
	}

	// create erow if no row had the index visible
	if len(flash) == 0 {
		erow, err := info.NewERow(rowPos)
		if err != nil {
			ed.Error(err)
			return
		}
		flash[erow] = true
		setindex[erow] = true
	}

	// flash rows positions
	for e := range flash {
		if setindex[e] {
			e.Row.TextArea.TextCursor.SetIndex(index)
		}
		e.MakeIndexVisibleAndFlash(index)
	}
}

func openERowFileOrNew(ed *Editor, fp *parseutil.FilePos, rowPos *ui.RowPos) {
	// no line/col
	if fp.Line != 0 {
		panic("line not zero")
	}

	info := ed.ReadERowInfo(fp.Filename)

	// create new row
	if len(info.ERows) == 0 {
		_, err := info.NewERow(rowPos)
		if err != nil {
			ed.Error(err)
			return
		}
	}

	//  flash rows names
	for _, e := range info.ERows {
		e.Flash()
	}
}

//----------

func OpenERowFileOffsetVisibleOrNew(ed *Editor, fo *parseutil.FileOffset, rowPos *ui.RowPos) {

	info := ed.ReadERowInfo(fo.Filename)

	flash := map[*ERow]bool{}

	// add rows that have the index already visible to be flashed
	for _, e := range info.ERows {
		if e.Row.TextArea.IsRangeVisible(fo.Offset, fo.Len) {
			flash[e] = true
		}
	}

	// create erow if no row had the index visible
	if len(flash) == 0 {
		erow, err := info.NewERow(rowPos)
		if err != nil {
			ed.Error(err)
			return
		}
		flash[erow] = true
	}

	// flash rows positions
	for e := range flash {
		e.Row.TextArea.MakeRangeVisible(fo.Offset, fo.Len)
		e.Row.TextArea.FlashIndexLen(fo.Offset, fo.Len)
	}
}
