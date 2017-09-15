package core

import (
	"fmt"
	"os"

	"github.com/jmigpin/editor/core/cmdutil"
	"github.com/jmigpin/editor/core/contentcmd"
	"github.com/jmigpin/editor/core/toolbardata"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/xgbutil"
	"github.com/pkg/errors"
)

type ERow struct {
	ed               *Editor
	row              *ui.Row
	tbsd             *toolbardata.StringData
	decodedPart0Arg0 string
	fi               struct {
		doneFirst bool
		fileInfo  os.FileInfo
		err       error // error while getting fileinfo, if any
	}
}

func NewERow(ed *Editor, row *ui.Row, tbStr string) *ERow {
	erow := &ERow{ed: ed, row: row}

	// set toolbar before setting event handlers
	row.Toolbar.SetStrClear(tbStr, true, true)

	erow.initHandlers()

	// run after event handlers are set
	erow.parseToolbar()

	return erow
}
func (erow *ERow) initHandlers() {
	row := erow.row
	ed := erow.ed
	// toolbar set str
	row.Toolbar.EvReg.Add(ui.TextAreaSetStrEventId,
		&xgbutil.ERCallback{func(ev0 interface{}) {
			erow.parseToolbar()
		}})
	// toolbar cmds
	row.Toolbar.EvReg.Add(ui.TextAreaCmdEventId,
		&xgbutil.ERCallback{func(ev0 interface{}) {
			ToolbarCmdFromRow(erow)
		}})
	// textarea set str
	row.TextArea.EvReg.Add(ui.TextAreaSetStrEventId,
		&xgbutil.ERCallback{func(ev0 interface{}) {
			// edited
			_, fi, err := erow.FileInfo()
			if err == nil && !fi.IsDir() {
				erow.row.Square.SetValue(ui.SquareEdited, true)
			}
		}})
	// textarea content cmds
	row.TextArea.EvReg.Add(ui.TextAreaCmdEventId,
		&xgbutil.ERCallback{func(ev0 interface{}) {
			contentcmd.Cmd(erow)
		}})
	// textarea error
	row.TextArea.EvReg.Add(ui.TextAreaErrorEventId,
		&xgbutil.ERCallback{func(ev0 interface{}) {
			err := ev0.(error)
			ed.Error(err)
		}})
	// close
	row.EvReg.Add(ui.RowCloseEventId,
		&xgbutil.ERCallback{func(ev0 interface{}) {
			cmdutil.RowCtxCancel(row)
			ed.reopenRow.Add(row)
			erow.ed.fw.Remove(erow)
		}})
}
func (erow *ERow) Row() *ui.Row {
	return erow.row
}
func (erow *ERow) Ed() cmdutil.Editorer {
	return erow.ed
}
func (erow *ERow) DecodedPart0Arg0() string {
	return erow.decodedPart0Arg0
}
func (erow *ERow) parseToolbar() {
	erow.tbsd = toolbardata.NewStringData(erow.row.Toolbar.Str())

	// update toolbar with encoded value
	s := erow.tbsd.StrWithPart0Arg0Encoded()
	if s != erow.tbsd.Str {
		// set str will trigger event that parses again
		erow.Row().Toolbar.SetStrClear(s, false, false)
		// TODO: adjust cursor
		return
	}

	fp, ok := erow.tbsd.DecodePart0Arg0()
	if !ok {
		fp = ""
	}

	if erow.decodedPart0Arg0 == fp && erow.fi.doneFirst {
		return
	}
	erow.decodedPart0Arg0 = fp
	erow.fi.doneFirst = true

	if fp == "" || erow.ed.IsSpecialName(fp) {
		erow.ed.fw.Remove(erow)
	} else {
		erow.ed.fw.AddUpdate(erow, fp)
	}

	erow.updateFileInfo()
}

// Also called from FSNWatcher.
func (erow *ERow) updateFileInfo() {
	erow.fi.fileInfo = nil
	erow.fi.err = nil

	notExist := false
	defer func() {
		erow.row.Square.SetValue(ui.SquareNotExist, notExist)
	}()

	fp := erow.decodedPart0Arg0
	if fp == "" {
		erow.fi.err = fmt.Errorf("missing part0")
		return
	}

	if erow.ed.IsSpecialName(fp) {
		erow.fi.err = fmt.Errorf("special part0: %v", fp)
		return
	}

	fi, err := os.Stat(fp)
	if err != nil {
		erow.fi.err = errors.Wrapf(err, "updateinfo")
		if os.IsNotExist(err) {
			notExist = true
		}
		return
	}
	erow.fi.fileInfo = fi
}

func (erow *ERow) FileInfo() (string, os.FileInfo, error) {
	if erow.fi.err != nil {
		return "", nil, errors.Wrapf(erow.fi.err, "fileinfo")
	}
	return erow.decodedPart0Arg0, erow.fi.fileInfo, nil
}
func (erow *ERow) ToolbarSD() *toolbardata.StringData {
	return erow.tbsd
}

func (erow *ERow) LoadContentClear() error {
	return erow.loadContent(true)
}
func (erow *ERow) ReloadContent() error {
	return erow.loadContent(false)
}
func (erow *ERow) loadContent(clear bool) error {
	fp, _, err := erow.FileInfo()
	if err != nil {
		return err
	}
	content, err := filepathContent(fp)
	if err != nil {
		return errors.Wrapf(err, "loadcontent")
	}
	erow.row.TextArea.SetStrClear(content, clear, clear)
	erow.row.Square.SetValue(ui.SquareEdited, false)
	erow.row.Square.SetValue(ui.SquareDiskChanges, false)
	return nil
}
func (erow *ERow) SaveContent(str string) error {
	fp, fi, err := erow.FileInfo()
	if err == nil {
		if fi.IsDir() {
			return fmt.Errorf("saving a directory: %v", fp)
		}
	} else {
		// save non existing file
		fp2 := erow.decodedPart0Arg0
		if fp2 == "" {
			return fmt.Errorf("missing filename")
		}
		if erow.ed.IsSpecialName(fp2) {
			return fmt.Errorf("can't save special name: %s", fp2)
		}
		fp = fp2
	}
	err = erow.saveContent2(str, fp)
	if err != nil {
		return err
	}
	erow.row.Square.SetValue(ui.SquareEdited, false)
	erow.row.Square.SetValue(ui.SquareDiskChanges, false)
	return nil
}
func (erow *ERow) saveContent2(str string, filename string) error {
	// remove from file watcher to avoid events while writing
	erow.ed.fw.Remove(erow)
	// re-add through update file info (needed if file didn't exist)
	defer erow.updateFileInfo()

	// save
	flags := os.O_WRONLY | os.O_TRUNC | os.O_CREATE
	f, err := os.OpenFile(filename, flags, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	defer f.Sync()
	_, err = f.Write([]byte(str))
	return err
}

func (erow *ERow) TextAreaAppendAsync(str string) {
	erow.ed.ui.TextAreaAppendAsync(erow.row.TextArea, str)
}
