package edit

import (
	"errors"
	"os"

	"github.com/howeyc/fsnotify"
	"github.com/jmigpin/editor/edit/cmdutil"
	"github.com/jmigpin/editor/edit/contentcmd"
	"github.com/jmigpin/editor/edit/toolbardata"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/xutil/xgbutil"
)

type ERow struct {
	ed           *Editor
	row          *ui.Row
	tbsd         *toolbardata.StringData
	fileInfo     os.FileInfo
	fileInfoPath string
}

func NewERow(ed *Editor, row *ui.Row, tbStr string) *ERow {
	erow := &ERow{ed: ed, row: row}

	// set toolbar before setting event handlers
	row.Toolbar.SetStrClear(tbStr, true, true)

	erow.init()

	// run after event handlers are set
	erow.parseToolbar(erow.row.Toolbar.Str())

	return erow
}
func (erow *ERow) init() {
	row := erow.row
	ed := erow.ed
	// toolbar set str
	row.Toolbar.EvReg.Add(ui.TextAreaSetStrEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			erow.parseToolbar(erow.row.Toolbar.Str())
		}})
	// toolbar cmds
	row.Toolbar.EvReg.Add(ui.TextAreaCmdEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			ToolbarCmdFromRow(erow)
		}})
	// textarea set str
	row.TextArea.EvReg.Add(ui.TextAreaSetStrEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			// dirty
			_, fi, ok := erow.FileInfo()
			if ok && !fi.IsDir() {
				erow.row.Square.SetValue(ui.SquareDirty, true)
			}
		}})
	// textarea content cmds
	row.TextArea.EvReg.Add(ui.TextAreaCmdEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			contentcmd.Cmd(erow)
		}})
	// textarea error
	row.TextArea.EvReg.Add(ui.TextAreaErrorEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			err := ev0.(error)
			ed.Error(err)
		}})
	// close
	row.EvReg.Add(ui.RowCloseEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
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
func (erow *ERow) parseToolbar(str string) {
	erow.tbsd = toolbardata.NewStringData(str)

	// insert home tilde on first part
	if len(erow.tbsd.Parts) > 0 {
		s1 := erow.tbsd.Parts[0].Str
		s2 := toolbardata.InsertHomeTilde(s1)
		if s1 != s2 {
			s3 := s2 + str[len(s1):]
			// reparse
			str = s3
			erow.tbsd = toolbardata.NewStringData(str)

			// set str, it will come back to this func again through event callback
			erow.Row().Toolbar.SetStrClear(str, false, false)
			return
		}
	}

	// keep file info
	notExist := false
	fp := erow.tbsd.FirstPartFilepath()
	fi, err := os.Stat(fp)
	if err != nil {
		if os.IsNotExist(err) {
			notExist = true
		}
		erow.fileInfo = nil
		erow.fileInfoPath = ""
		erow.ed.fw.Remove(erow)
	} else {
		erow.fileInfo = fi
		erow.fileInfoPath = fp
		if !fi.IsDir() {
			erow.ed.fw.Add(erow, fp)
		}
	}
	erow.row.Square.SetValue(ui.SquareNotExist, notExist)
}
func (erow *ERow) ToolbarSD() *toolbardata.StringData {
	return erow.tbsd
}
func (erow *ERow) FileInfo() (string, os.FileInfo, bool) {
	if erow.fileInfo == nil {
		return "", nil, false
	}
	return erow.fileInfoPath, erow.fileInfo, true
}

func (erow *ERow) LoadContentClear() error {
	return erow.loadContent(true)
}
func (erow *ERow) ReloadContent() error {
	return erow.loadContent(false)
}
func (erow *ERow) loadContent(clear bool) error {
	fp, _, ok := erow.FileInfo()
	if !ok {
		return errors.New("missing fileinfo")
	}
	content, err := filepathContent(fp)
	if err != nil {
		return err
	}
	erow.row.TextArea.SetStrClear(content, clear, clear)
	erow.row.Square.SetValue(ui.SquareDirty, false)
	erow.row.Square.SetValue(ui.SquareCold, false)
	return nil
}
func (erow *ERow) SaveContent(str string) error {
	_, fi, ok := erow.FileInfo()
	if !ok {
		return errors.New("fileinfo missing")
	}
	if fi.IsDir() {
		return errors.New("can't save a directory")
	}
	err := erow.saveContent2(str)
	erow.row.Square.SetValue(ui.SquareDirty, false)
	erow.row.Square.SetValue(ui.SquareCold, false)
	return err
}
func (erow *ERow) saveContent2(str string) error {
	// disable/enable file watcher to avoid events while writing
	erow.ed.fw.Remove(erow)
	defer erow.ed.fw.Add(erow, erow.fileInfoPath)

	//log.SetFlags(0)

	//u := path.Base(erow.fileInfoPath)
	//log.Printf("save content: %v", u)
	//defer log.Printf("save content done: %v", u)

	// save
	filename := erow.fileInfoPath
	flags := os.O_WRONLY | os.O_TRUNC | os.O_CREATE
	f, err := os.OpenFile(filename, flags, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	defer f.Sync()
	_, err = f.Write([]byte(str))

	//ed := erow.ed
	//ed.fw.m.Lock()
	//for _, v := range ed.fw.m.m {
	//log.Printf("** %v\n", path.Base(v))
	//}
	//ed.fw.m.Unlock()

	return err
}

// Directly Called by the editor fileswatcher - async.
func (erow *ERow) OnFilesWatcherEvent(ev *fsnotify.FileEvent) {

	//u := path.Base(ev.Name)
	//log.Printf("fw event: %v", u)
	//defer log.Printf("fw event done: %v", u)

	//if ev.IsModify() {
	//log.Printf("fw is modify: %v", ev.Name)
	//}

	sq := erow.row.Square
	if sq.Value(ui.SquareCold) == false {
		erow.row.Square.SetValue(ui.SquareCold, true)
		erow.ed.UI().RequestTreePaint()
	}
}

func (erow *ERow) TextAreaAppend(str string) {
	ta := erow.row.TextArea

	// cap max size
	maxSize := 1024 * 1024 * 10
	str = ta.Str() + str
	if len(str) > maxSize {
		d := len(str) - maxSize
		str = str[d:]
	}

	ta.SetStrClear(str, false, true) // clear undo for massive savings

}
