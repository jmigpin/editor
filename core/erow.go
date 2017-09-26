package core

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/jmigpin/editor/core/cmdutil"
	"github.com/jmigpin/editor/core/contentcmd"
	"github.com/jmigpin/editor/core/toolbardata"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/xgbutil"
	"github.com/pkg/errors"
)

type ERow struct {
	ed  *Editor
	row *ui.Row
	td  *toolbardata.ToolbarData

	state struct {
		name  string // decoded part0arg0
		tbStr string

		filename string // abs filename from name
		isDir    bool
		watch    bool
	}
}

func NewERow(ed *Editor, row *ui.Row, tbStr string) *ERow {
	erow := &ERow{ed: ed, row: row}

	// set toolbar before setting event handlers
	row.Toolbar.SetStrClear(tbStr, true, true)

	erow.initHandlers()

	// run after event handlers are set
	erow.parseToolbar(nil)

	return erow
}
func (erow *ERow) initHandlers() {
	row := erow.row
	ed := erow.ed
	// toolbar set str
	row.Toolbar.EvReg.Add(ui.TextAreaSetStrEventId,
		&xgbutil.ERCallback{func(ev0 interface{}) {
			erow.parseToolbar(ev0.(*ui.TextAreaSetStrEvent))
		}})
	// toolbar cmds
	row.Toolbar.EvReg.Add(ui.TextAreaCmdEventId,
		&xgbutil.ERCallback{func(ev0 interface{}) {
			ToolbarCmdFromRow(erow)
		}})
	// textarea set str
	row.TextArea.EvReg.Add(ui.TextAreaSetStrEventId,
		&xgbutil.ERCallback{func(ev0 interface{}) {
			if !erow.IsDir() && !erow.IsSpecialName() {
				erow.SetEdited(true)
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

			if erow.state.watch {
				erow.ed.fwatcher.Remove(erow.state.filename)
			}
		}})
}

func (erow *ERow) Row() *ui.Row {
	return erow.row
}
func (erow *ERow) Ed() cmdutil.Editorer {
	return erow.ed
}
func (erow *ERow) ToolbarData() *toolbardata.ToolbarData {
	return erow.td
}

func (erow *ERow) SetEdited(v bool) {
	erow.row.Square.SetValue(ui.SquareEdited, v)
}
func (erow *ERow) SetDiskChanges(v bool) {
	erow.row.Square.SetValue(ui.SquareDiskChanges, v)
}

func (erow *ERow) Name() string {
	return erow.state.name
}
func (erow *ERow) Filename() string {
	return erow.state.filename
}
func (erow *ERow) IsDir() bool {
	return erow.state.isDir
}
func (erow *ERow) Dir() string {
	fp := erow.Filename()
	if erow.IsDir() {
		return fp
	}
	return path.Dir(fp)
}

func (erow *ERow) IsSpecialName() bool {
	return erow.ed.IsSpecialName(erow.state.name)
}

func (erow *ERow) parseToolbar(ev *ui.TextAreaSetStrEvent) {
	tbStr := erow.Row().Toolbar.Str()
	td := toolbardata.NewToolbarData(tbStr, erow.ed.HomeVars())
	name := td.DecodePart0Arg0()

	// ev is nil on first call at erow creation.
	// don't allow changing the first part
	if ev != nil {
		if name != erow.state.name {
			ev.TextArea.SetRawStr(erow.state.tbStr)
			erow.Ed().Errorf("can't change toolbar first part")
			return
		}
	}

	erow.td = td
	erow.state.tbStr = tbStr
	erow.state.name = name

	// update toolbar with encoded value
	s := td.StrWithPart0Arg0Encoded()
	if s != td.Str {
		// set str will trigger event that parses the toolbar again
		erow.Row().Toolbar.SetStrClear(s, false, false)
		return
	}

	erow.UpdateState()
}

func (erow *ERow) UpdateState() {
	prev := erow.state

	erow.updateFileinfo()

	cur := &erow.state
	if prev == *cur {
		return
	}

	// stop watching previous
	if prev.watch && prev.filename != cur.filename {
		erow.ed.fwatcher.Remove(prev.filename)
	}

	// start watching current
	cur.watch = false
	if cur.filename != "" && !erow.ed.IsSpecialName(cur.filename) {
		cur.watch = true
		erow.ed.fwatcher.Add(cur.filename)
	}
}

func (erow *ERow) updateFileinfo() {
	c := &erow.state
	c.filename = ""
	c.isDir = false

	if erow.ed.IsSpecialName(c.name) {
		return
	}

	abs, err := filepath.Abs(c.name)
	if err == nil {
		c.filename = abs

		fi, err := os.Stat(c.filename)
		if err == nil {
			if fi.IsDir() {
				c.isDir = true
			}
		}
	}
}

func (erow *ERow) LoadContentClear() error {
	return erow.loadContent(true)
}
func (erow *ERow) ReloadContent() error {
	return erow.loadContent(false)
}
func (erow *ERow) loadContent(clear bool) error {
	if erow.IsSpecialName() {
		return fmt.Errorf("can't load special name: %s", erow.state.name)
	}
	fp := erow.Filename()
	content, err := erow.filepathContent(fp)
	if err != nil {
		return errors.Wrapf(err, "loadcontent")
	}
	erow.row.TextArea.SetStrClear(content, clear, clear)
	erow.SetEdited(false)
	erow.SetDiskChanges(false)
	return nil
}
func (erow *ERow) SaveContent(str string) error {
	if erow.IsSpecialName() {
		return fmt.Errorf("can't save special name: %s", erow.state.name)
	}
	fp := erow.Filename()
	if erow.IsDir() {
		return fmt.Errorf("can't save a directory: %v", fp)
	}
	err := erow.saveContent2(str, fp)
	if err != nil {
		return err
	}
	erow.SetEdited(false)
	erow.SetDiskChanges(false)
	return nil
}
func (erow *ERow) saveContent2(str string, filename string) error {
	// remove from file watcher to avoid events while writing
	erow.state.watch = false
	erow.ed.fwatcher.Remove(erow.state.filename)

	// re-add through update state (needed if file didn't exist)
	defer erow.UpdateState()

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

func (erow *ERow) filepathContent(filepath string) (string, error) {
	fi, err := os.Stat(filepath)
	if err != nil {
		return "", err
	}
	if fi.IsDir() {
		return cmdutil.ListDir(filepath, false, true)
	}
	// file content
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
