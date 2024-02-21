package core

import (
	"errors"

	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/parseutil"
)

type OpenFileERowConfig struct {
	FilePos *parseutil.FilePos
	RowPos  *ui.RowPos

	CancelIfExistent      bool
	NewIfNotExistent      bool
	NewIfOffsetNotVisible bool

	//FlashRowsIfNotFlashed bool
	FlashVisibleOffsets bool // flashes rows if not visible

	PreferedERow *ERow
}

// TODO: make it UI safe? rename to openfileerowasync?
func OpenFileERow(ed *Editor, conf *OpenFileERowConfig) {
	if _, err := openFileERow2(ed, conf); err != nil {
		ed.Error(err)
	}
}

func openFileERow2(ed *Editor, conf *OpenFileERowConfig) (isNew bool, _ error) {
	// filename
	var filename string
	if conf.FilePos != nil {
		filename = conf.FilePos.Filename
	} else if conf.PreferedERow != nil {
		filename = conf.PreferedERow.Info.Name()
	} else {
		return false, errors.New("missing filename")
	}

	info := ed.ReadERowInfo(filename)

	// do nothing if existent
	if conf.CancelIfExistent && len(info.ERows) > 0 {
		return false, nil
	}

	createNew := false

	// helper func: cache for LineColumnIndex
	lciVal := 0
	lciDone := false
	cacheLineColumnIndex := func(rd iorw.ReaderAt) int {
		if lciDone {
			return lciVal
		}
		lciDone = true
		if conf.FilePos.Line == 0 { // missing line/col, should be ">=1"
			lciVal = -1
		} else {
			u, err := parseutil.LineColumnIndex(rd, conf.FilePos.Line, conf.FilePos.Column)
			if err != nil {
				lciVal = -1
			} else {
				lciVal = u
			}
		}
		return lciVal
	}

	// helper func: get offset
	getOffset := func() int {
		if conf.FilePos != nil {
			if conf.FilePos.HasNoLinecol() {
				return conf.FilePos.Offset
			}
			if erow0, ok := info.FirstERow(); ok {
				ta := erow0.Row.TextArea
				return cacheLineColumnIndex(ta.RW())
			}
		}
		return -1
	}

	// should create new if not existent
	if conf.NewIfNotExistent {
		if len(info.ERows) == 0 {
			createNew = true
		}
	}

	// should create new if offset not visible
	if conf.NewIfOffsetNotVisible {
		if !createNew {
			if len(info.ERows) == 0 {
				createNew = true
			} else {
				offset := getOffset()
				if offset >= 0 {
					visible := false
					for _, e := range info.ERows {
						if e.Row.TextArea.IndexVisible(offset) {
							visible = true
							break
						}
					}
					createNew = !visible
				}
			}
		}
	}

	// create new erow
	var newERow *ERow
	if createNew {
		isNew = true
		if conf.RowPos == nil {
			return isNew, errors.New("missing row position")
		}
		erow, err := NewLoadedERow(info, conf.RowPos)
		if err != nil {
			return isNew, err
		}
		newERow = erow
	}

	// make offset visible
	flashed := make(map[*ERow]bool)
	offset := getOffset()
	if offset >= 0 {
		if len(info.ERows) == 0 {
			return isNew, errors.New("missing erow to make offset visible")
		}

		// use newly created erow
		erow := newERow
		// use prefered erow
		if erow == nil {
			for _, e := range info.ERows {
				if e == conf.PreferedERow {
					erow = e
				}
			}
		}
		// use existing row with visible offset
		if erow == nil {
			for _, e := range info.ERows {
				if e.Row.TextArea.IndexVisible(offset) {
					erow = e
					break
				}
			}
		}
		// use first row in UI order
		if erow == nil {
			erow = info.ERowsInUIOrder()[0]
		}

		// setup chosen erow
		//erow.Row.EnsureTextAreaMinimumHeight()
		erow.Row.EnsureOneToolbarLineYVisible()
		erow.Row.TextArea.Cursor().SetIndexSelectionOff(offset)
		erow.Row.TextArea.MakeIndexVisible(offset)

		// flash visible offsets
		if conf.FlashVisibleOffsets {
			o, l := 0, 0
			if conf.FilePos != nil {
				if conf.FilePos.HasNoLinecol() {
					o, l = conf.FilePos.Offset, conf.FilePos.Len
				} else {
					o = offset
				}
			}

			for _, e := range info.ERows {
				if e.Row.TextArea.IndexVisible(offset) {
					e.MakeRangeVisibleAndFlash(o, l)
					flashed[e] = true
				}
			}
		}

		// no flashes were done (no index visible, small rows), flash target
		if len(flashed) == 0 {
			erow.Flash()
			flashed[erow] = true
		}
	}

	// flash rows if not flashed already
	//if conf.FlashRowsIfNotFlashed ||
	if conf.FlashVisibleOffsets && offset < 0 {
		for _, e := range info.ERows {
			if !flashed[e] {
				e.Flash()
			}
		}
	}

	return isNew, nil
}
