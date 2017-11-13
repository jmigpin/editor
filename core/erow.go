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
	"github.com/jmigpin/editor/uiutil/event"
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

	return erow
}
func (erow *ERow) initHandlers() {
	row := erow.row
	ed := erow.ed
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
		erow.UpdateState()
		erow.UpdateDuplicates()
	})
	// textarea content cmds
	row.TextArea.EvReg.Add(ui.TextAreaCmdEventId, func(ev0 interface{}) {
		contentcmd.Cmd(erow)
	})
	// key shortcuts
	row.EvReg.Add(ui.RowInputEventId, erow.onRowInput)
	// close
	row.EvReg.Add(ui.RowCloseEventId, func(ev0 interface{}) {
		cmdutil.RowCtxCancel(row)
		ed.reopenRow.Add(row)
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

	erow.UpdateState()
}

func (erow *ERow) UpdateState() {
	prev := erow.state // copy

	erow.updateState2()

	// increase watch before decreasing, will not remove the watch if it was added before
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
		erow.row.Square.SetValue(ui.SquareEdited, edited)
	}

	// disk changes
	if erow.IsRegular() {
		changes := !bytes.Equal(erow.state.disk.hash, erow.saved.hash)
		erow.row.Square.SetValue(ui.SquareDiskChanges, changes)
	}

	// not exist
	erow.row.Square.SetValue(ui.SquareNotExist, erow.state.notExist)

	// has duplicates
	hasDuplicate := len(erow.duplicateERows()) >= 2
	erow.row.Square.SetValue(ui.SquareDuplicate, hasDuplicate)
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
	return erow.loadContent(true)
}
func (erow *ERow) ReloadContent() error {
	return erow.loadContent(false)
}
func (erow *ERow) loadContent(clear bool) error {
	if erow.IsSpecialName() {
		return fmt.Errorf("can't load special name: %s", erow.Name())
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

	erow.disableTextAreaSetStrEventHandler = true // avoid running UpdateDuplicates twice
	erow.row.TextArea.SetStrClear(str, clear, clear)
	erow.disableTextAreaSetStrEventHandler = false

	erow.UpdateState()
	erow.UpdateDuplicates()

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

	// There is no need to update state or duplicates, the file watcher would emit event.
	// Here for redundancy to avoid having erow state be depended on the file watcher
	// in edge cases like waking up from hibernation and things not working properly.

	erow.UpdateState()
	erow.UpdateDuplicates()

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
	erow.ed.ui.EnqueueRunFunc(func() {
		ta := erow.row.TextArea

		// max size for appends
		maxSize := 5 * 1024 * 1024
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
		case evt.Modifiers.Is(event.ModControl) && evt.Code == ' ':
			//fb := erow.row.FloatBox()
			//fb.SetStr("testing")
		}
	}
}

func (erow *ERow) UpdateDuplicates() {
	//log.Printf("update duplicates %p", erow)

	for _, erow2 := range erow.duplicateERows() {
		if erow == erow2 {
			continue // otherwise will discard own history
		}

		ta := erow.Row().TextArea
		ta2 := erow2.Row().TextArea

		ci := ta2.CursorIndex()
		ip := ta2.GetPoint(ci)

		// use temporary history to set the string in the duplicate
		// then get the main shared history to allow undo/redo

		tmp := tahistory.NewHistory(1)
		ta2.SetHistory(tmp)

		erow2.disableTextAreaSetStrEventHandler = true // avoid recursive events
		ta2.SetStrClear(ta.Str(), false, false)
		erow2.disableTextAreaSetStrEventHandler = false

		ta2.SetHistory(ta.History())

		// restore position (avoid cursor moving while editing in another row)
		pi := ta2.GetIndex(&ip)
		ta2.SetCursorIndex(pi)

		erow2.saved.size = erow.saved.size
		erow2.saved.hash = erow.saved.hash
		erow2.UpdateState()
	}
}
func (erow *ERow) duplicateERows() []*ERow {
	if !erow.IsRegular() {
		return nil
	}
	// includes self
	var u []*ERow
	for _, erow2 := range erow.ed.erows {
		if erow2.Filename() == erow.Filename() {
			u = append(u, erow2)
		}
	}
	return u
}
