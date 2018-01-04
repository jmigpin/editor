package core

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/jmigpin/editor/core/cmdutil"
	"github.com/jmigpin/editor/core/contentcmd"
	"github.com/jmigpin/editor/core/toolbardata"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/ui/tautil/tahistory"
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/pkg/errors"
)

type ERow struct {
	ed  *Editor
	row *ui.Row
	td  *toolbardata.ToolbarData

	nameIsSet bool
	name      string // decoded part0arg0
	tbStr     string

	state ERowState

	saved struct {
		size int64
		hash []byte
	}

	disableTextAreaSetStrEventHandler bool
}

type ERowState struct {
	filename  string // abs filename from name
	isRegular bool
	isDir     bool
	notExist  bool

	watch bool

	disk struct {
		modTime time.Time
		hash    []byte
	}
}

func NewERow(ed *Editor, row *ui.Row, tbStr string) *ERow {
	erow := &ERow{ed: ed, row: row}

	// set toolbar before setting event handlers
	row.Toolbar.SetStrClear(tbStr, true, true)

	erow.initHandlers()

	// run after event handlers are set
	erow.parseToolbar()

	erow.UpdateState()
	erow.UpdateDuplicatesState()

	erow.setupTextAreaCommentString()

	return erow
}
func (erow *ERow) initHandlers() {
	row := erow.row

	erow.ed.RegisterERow(erow)

	// toolbar set str
	row.Toolbar.EvReg.Add(ui.TextAreaSetStrEventId, func(ev0 interface{}) {
		erow.parseToolbar()
	})
	// toolbar cmds
	row.Toolbar.EvReg.Add(ui.TextAreaCmdEventId, func(ev0 interface{}) {
		ToolbarCmdFromRow(erow.ed, erow)
	})
	// textarea set str
	row.TextArea.EvReg.Add(ui.TextAreaSetStrEventId, func(ev0 interface{}) {
		if erow.disableTextAreaSetStrEventHandler {
			return
		}
		erow.UpdateStateAndDuplicates()
	})
	// textarea content cmds
	row.TextArea.EvReg.Add(ui.TextAreaCmdEventId, func(ev0 interface{}) {
		ev := ev0.(*ui.TextAreaCmdEvent)
		contentcmd.Cmd(erow, ev.Index)
	})
	// key shortcuts
	row.EvReg.Add(ui.RowInputEventId, erow.onRowInput)
	// close
	row.EvReg.Add(ui.RowCloseEventId, func(ev0 interface{}) {
		erow.ed.UnregisterERow(erow)
		erow.UpdateStateAndDuplicates()
		cmdutil.RowCtxCancel(row)
		erow.ed.reopenRow.Add(row)
		if erow.state.watch {
			erow.ed.DecreaseWatch(erow.state.filename)
		}
	})
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

func (erow *ERow) Name() string {
	return erow.name
}
func (erow *ERow) Filename() string {
	return erow.state.filename
}
func (erow *ERow) IsRegular() bool {
	return erow.state.isRegular
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
	return erow.ed.IsSpecialName(erow.name)
}

func (erow *ERow) parseToolbar() {
	tbStr := erow.Row().Toolbar.Str()
	td := toolbardata.NewToolbarData(tbStr, erow.ed.HomeVars())

	// update toolbar with encoded value
	s := td.StrWithPart0Arg0Encoded()
	if s != tbStr {
		// will trigger event that will parse toolbar again
		erow.Row().Toolbar.SetStrClear(s, false, false)
		return
	}

	name := td.DecodePart0Arg0()

	// don't allow changing the first part
	if erow.nameIsSet && name != erow.name {
		erow.Ed().Errorf("can't change toolbar first part")
		// will trigger event that will parse toolbar again
		erow.Row().Toolbar.SetStrClear(erow.tbStr, false, false)
		return
	}

	erow.td = td
	erow.tbStr = tbStr
	erow.nameIsSet = true
	erow.name = name
}

func (erow *ERow) UpdateState() {
	prev := erow.state // copy

	erow.updateState2()

	// increase watch before decreasing to prevent removing the watch if it was added before
	if erow.state.watch {
		erow.ed.IncreaseWatch(erow.state.filename)
	}
	if prev.watch {
		erow.ed.DecreaseWatch(prev.filename)
	}

	// edited
	if erow.IsRegular() {
		str := erow.Row().TextArea.Str()
		edited := int64(len(str)) != erow.saved.size
		if !edited {
			hash := erow.contentHash([]byte(str))
			edited = !bytes.Equal(hash, erow.saved.hash)
		}
		erow.row.SetState(ui.EditedRowState, edited)
	}

	// disk changes
	if erow.IsRegular() {
		changes := !bytes.Equal(erow.state.disk.hash, erow.saved.hash)
		erow.row.SetState(ui.DiskChangesRowState, changes)
	}

	// not exist
	erow.row.SetState(ui.NotExistRowState, erow.state.notExist)

	// has duplicates
	hasDuplicate := len(erow.duplicateERows()) >= 2
	if erow.IsRegular() {
		erow.row.SetState(ui.DuplicateRowState, hasDuplicate)
	}
	erow.updateHighlightDuplicates(hasDuplicate)
}

func (erow *ERow) updateState2() {
	prev := erow.state // copy

	// reset state
	erow.state = ERowState{}

	if erow.ed.IsSpecialName(erow.name) {
		return
	}

	st := &erow.state

	abs, err := filepath.Abs(erow.name)
	if err != nil {
		return
	}
	st.filename = abs

	// need to watch even if it doesn't exist (can be created later)
	st.watch = true

	fi, err := os.Stat(st.filename)
	st.notExist = os.IsNotExist(err)
	if err != nil {
		return
	}

	st.isRegular = fi.Mode().IsRegular()
	st.isDir = fi.IsDir()

	st.disk.modTime = fi.ModTime()

	// update disk hash only if the modified time has changed
	if st.disk.modTime.Equal(prev.disk.modTime) {
		erow.state.disk.hash = prev.disk.hash
	} else {
		b, err := ioutil.ReadFile(st.filename)
		if err == nil {
			erow.state.disk.hash = erow.contentHash(b)
		}
	}
}

func (erow *ERow) contentHash(b []byte) []byte {
	h := sha1.New()
	h.Write(b)
	return h.Sum(nil)
}

func (erow *ERow) LoadContentClear() error {
	return erow.loadContent(false, true)
}
func (erow *ERow) ReloadContent() error {
	return erow.loadContent(true, false)
}
func (erow *ERow) loadContent(reload, clear bool) error {
	if erow.IsSpecialName() {
		return fmt.Errorf("can't load special name: %s", erow.Name())
	}

	// if it has duplicates, load content from another row
	if !reload && erow.IsRegular() {
		erows := erow.duplicateERows()
		if len(erows) >= 2 {
			for _, e := range erows {
				if e == erow {
					continue
				}
				//e.UpdateDuplicatesContent()
				//e.UpdateDuplicatesState()
				erow.updateContentFromDuplicate(e)
				erow.UpdateState()
				return nil
			}
		}
	}

	fp := erow.Filename()
	str, err := erow.filepathContent(fp)
	if err != nil {
		return errors.Wrapf(err, "loadcontent")
	}

	if erow.IsRegular() {
		erow.saved.size = int64(len(str))
		erow.saved.hash = erow.contentHash([]byte(str))
	}

	erow.disableTextAreaSetStrEventHandler = true // avoid recursive events
	erow.row.TextArea.SetStrClear(str, clear, clear)
	erow.disableTextAreaSetStrEventHandler = false

	erow.UpdateStateAndDuplicates()

	return nil
}
func (erow *ERow) SaveContent(str string) error {
	if erow.IsSpecialName() {
		return fmt.Errorf("can't save special name: %s", erow.Name())
	}
	fp := erow.Filename()
	if erow.IsDir() {
		return fmt.Errorf("can't save a directory: %v", fp)
	}
	err := erow.saveContent2(str, fp)
	if err != nil {
		return err
	}

	erow.saved.size = int64(len(str))
	erow.saved.hash = erow.contentHash([]byte(str))

	erow.UpdateStateAndDuplicates()

	return nil
}
func (erow *ERow) saveContent2(str string, filename string) error {
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

func (erow *ERow) filepathContent(filepath string) (string, error) {
	if erow.IsDir() {
		return cmdutil.ListDir(filepath, false, true)
	}
	// file content
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (erow *ERow) TextAreaAppendAsync(str string) {
	erow.ed.ui.RunOnUIThread(func() {
		ta := erow.row.TextArea

		// max size for appends
		// TODO: unlimited output? some xterms have more or less this limit as well. Bigger limits will slow down the ui since it will be calculating the new string content. This will be improved once the textarea drawer supports append/cutTop operations.
		maxSize := 64 * 1024
		str2 := ta.Str() + str
		if len(str2) > maxSize {
			d := len(str2) - maxSize
			str2 = str2[d:]
		}

		// false,true = keep pos, but clear undo for massive savings
		ta.SetStrClear(str2, false, true)
	})
}

func (erow *ERow) onRowInput(ev0 interface{}) {
	ev := ev0.(*ui.RowInputEvent)
	switch evt := ev.Event.(type) {
	case *event.KeyDown:
		switch {
		case evt.Modifiers.Is(event.ModControl) && evt.Code == 's':
			cmdutil.SaveRowFile(erow)
		case evt.Modifiers.Is(event.ModControl) && evt.Code == 'f':
			cmdutil.FindShortcut(erow)
		}
	case *event.MouseEnter:
		erow.updateHighlightDuplicates(true)
	case *event.MouseLeave:
		erow.updateHighlightDuplicates(false)
	}
}

func (erow *ERow) updateHighlightDuplicates(v bool) {
	erows := erow.duplicateERows()
	if !v || (v && len(erows) >= 2) {
		for _, e := range erows {
			e.Row().SetState(ui.HighlightDuplicateRowState, v)
		}
	}
}

func (erow *ERow) UpdateStateAndDuplicates() {
	erow.UpdateState()
	erow.UpdateDuplicatesContent()
	erow.UpdateDuplicatesState()
}

func (erow *ERow) UpdateDuplicatesState() {
	for _, e := range erow.duplicateERows() {
		if e == erow {
			continue
		}
		e.UpdateState()
	}
}
func (erow *ERow) UpdateDuplicatesContent() {
	for _, e := range erow.duplicateERows() {
		if e == erow {
			continue
		}
		e.updateContentFromDuplicate(erow)
	}
}
func (erow *ERow) updateContentFromDuplicate(srcERow *ERow) {
	// don't update itself - it will erase its own history
	if erow == srcERow {
		return
	}

	// update duplicate content only for files
	if !erow.IsRegular() {
		return
	}

	srcTa := srcERow.Row().TextArea
	ta := erow.Row().TextArea

	ci := ta.CursorIndex()
	ip := ta.GetPoint(ci)

	// use temporary history to set the string in the duplicate
	// then get the main shared history to allow undo/redo

	tmp := tahistory.NewHistory(1)
	ta.SetHistory(tmp)

	erow.disableTextAreaSetStrEventHandler = true // avoid recursive events
	ta.SetStrClear(srcTa.Str(), false, false)
	erow.disableTextAreaSetStrEventHandler = false

	ta.SetHistory(srcTa.History())

	// restore position (avoid cursor moving while editing in another row)
	pi := ta.GetIndex(&ip)
	ta.SetCursorIndex(pi)

	erow.saved.size = srcERow.saved.size
	erow.saved.hash = srcERow.saved.hash
}

func (erow *ERow) duplicateERows() []*ERow {
	return erow.ed.FindERows(erow.Filename())
}

func (erow *ERow) Flash() {
	p, ok := erow.td.GetPartAtIndex(0)
	if ok {
		tok := &p.Token

		// accurate using arg0
		if len(p.Args) > 0 {
			tok = p.Args[0]
		}

		erow.Row().Toolbar.FlashIndexLen(tok.S, tok.E-tok.S)
	}
}

func (erow *ERow) setupTextAreaCommentString() {
	cstr := "//"
	switch filepath.Ext(erow.Filename()) {
	case "", ".sh", ".conf", ".list", ".txt":
		cstr = "#"
	case ".go", ".c", ".cpp", ".h", ".hpp":
		cstr = "//"
	}
	erow.row.TextArea.CommentStr = cstr
}
