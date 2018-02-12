package core

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"
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

	disableTextAreaSetStrEventHandler bool

	execState struct {
		sync.Mutex
		ctx    context.Context
		cancel context.CancelFunc
	}
}

type ERowState struct {
	filename  string // abs filename from name
	isRegular bool
	isDir     bool
	notExist  bool

	watch bool

	// saved known hash
	savedHash struct {
		size int
		hash []byte
	}

	// real disk hash that could have been changed by another application
	diskHash struct {
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

	erow.updateStateAndDuplicates(true)

	// setup comment types after knowing filename extension
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

		// update godebug annotations if hash doesn't match
		cmdutil.DefaultGoDebugCmd.NakedUpdateERowAnnotations(erow)
	})
	// textarea content cmds
	row.TextArea.EvReg.Add(ui.TextAreaCmdEventId, func(ev0 interface{}) {
		ev := ev0.(*ui.TextAreaCmdEvent)
		contentcmd.Cmd(erow, ev.Index)
	})
	// textarea annotation click
	row.TextArea.EvReg.Add(ui.TextAreaAnnotationClickEventId, func(ev0 interface{}) {
		// NOTE: other modifiers might have been needed to trigger the event
		ev := ev0.(*ui.TextAreaAnnotationClickEvent)
		gcmd := &cmdutil.DefaultGoDebugCmd
		switch ev.Button {
		case event.ButtonLeft:
			gcmd.PrintAnnotation(erow, ev.Index, ev.IndexOffset)
		case event.ButtonWheelUp:
			gcmd.PreviousAnnotation(erow, ev.Index)
		case event.ButtonWheelDown:
			gcmd.NextAnnotation(erow, ev.Index)
		}
	})
	// key shortcuts
	row.EvReg.Add(ui.RowInputEventId, erow.onRowInput)
	// close
	row.EvReg.Add(ui.RowCloseEventId, func(ev0 interface{}) {
		erow.ed.UnregisterERow(erow)
		erow.UpdateStateAndDuplicates()
		erow.StopExecState()
		if !erow.IsSpecialName() {
			erow.ed.reopenRow.Add(row)
		}
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

func (erow *ERow) updateState() {
	prev := erow.state // copy

	erow.updateState2()

	// increase watch before decreasing to prevent removing the watch if it was added before
	if erow.state.watch {
		erow.ed.IncreaseWatch(erow.state.filename)
	}
	if prev.watch {
		erow.ed.DecreaseWatch(prev.filename)
	}
}

func (erow *ERow) updateState2() {
	prev := erow.state // copy

	// reset state
	erow.state = ERowState{}

	if erow.ed.IsSpecialName(erow.name) {
		return
	}

	st := &erow.state

	// always keep previous saved hash
	st.savedHash = prev.savedHash

	// absolute filename
	abs, err := filepath.Abs(erow.name)
	if err != nil {
		return
	}
	st.filename = abs

	// need to watch even if it doesn't exist (can be created later)
	st.watch = true

	// filename exists
	fi, err := os.Stat(st.filename)
	st.notExist = os.IsNotExist(err)
	if err != nil {
		return
	}

	st.isRegular = fi.Mode().IsRegular()
	st.isDir = fi.IsDir()

	st.diskHash.modTime = fi.ModTime()

	// disk hash: update only if the modified time has changed
	if st.diskHash.modTime.Equal(prev.diskHash.modTime) {
		erow.state.diskHash.hash = prev.diskHash.hash
	} else {
		b, err := ioutil.ReadFile(st.filename)
		if err == nil {
			erow.state.diskHash.hash = erow.contentHash(b)
		}
	}
}

func (erow *ERow) UpdateStateUI() {
	// edited
	edited := false
	if erow.IsRegular() {
		u := &erow.state.savedHash
		edited = !erow.TextAreaStrHashEqual(u.size, u.hash)
	}
	erow.row.SetState(ui.EditedRowState, edited)

	// annotations
	hasAnnotations := false
	hasAnnotationsEdited := false
	if erow.IsRegular() {
		di := cmdutil.GoDebugDataIndex()
		if di != nil {
			afd := di.AnnotatorFileData(erow.Filename())
			if afd != nil {
				hasAnnotations = true
				edited := !erow.TextAreaStrHashEqual(afd.FileSize, afd.FileHash)
				hasAnnotationsEdited = edited
			}
		}
	}
	erow.row.SetState(ui.AnnotationsRowState, hasAnnotations)
	erow.row.SetState(ui.AnnotationsEditedRowState, hasAnnotationsEdited)

	// disk changes
	changes := false
	if erow.IsRegular() {
		changes = !bytes.Equal(erow.state.diskHash.hash, erow.state.savedHash.hash)
	}
	erow.row.SetState(ui.DiskChangesRowState, changes)

	// not exist
	erow.row.SetState(ui.NotExistRowState, erow.state.notExist)

	// has duplicate
	hasDuplicate := false
	if erow.IsRegular() {
		hasDuplicate = len(erow.Duplicates()) >= 2
	}
	erow.row.SetState(ui.DuplicateRowState, hasDuplicate)
	// un-highlight duplicates
	if !hasDuplicate {
		erow.updateDuplicatesHighlight(false)
	}
}

func (erow *ERow) TextAreaStrHashEqual(size int, hash []byte) bool {
	str := erow.Row().TextArea.Str()
	if len(str) != size {
		return false
	}
	hash2 := erow.contentHash([]byte(str))
	return bytes.Equal(hash2, hash)
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
		otherDups := erow.OtherDuplicates()
		if len(otherDups) > 0 {
			otherDups[0].UpdateStateAndDuplicates()
			return nil
		}
	}

	fp := erow.Filename()
	str, err := erow.filepathContent(fp)
	if err != nil {
		return errors.Wrapf(err, "loadcontent")
	}

	if erow.IsRegular() {
		erow.state.savedHash.size = len(str)
		erow.state.savedHash.hash = erow.contentHash([]byte(str))
	}

	erow.disableTextAreaSetStrEventHandler = true // avoid recursive events
	erow.row.TextArea.SetStrClear(str, clear, clear)
	erow.disableTextAreaSetStrEventHandler = false

	erow.UpdateStateAndDuplicates()

	// init/update godebug annotations (could be recalling updatestateandduplicates)
	cmdutil.DefaultGoDebugCmd.NakedUpdateERowAnnotations(erow)

	return nil
}

func (erow *ERow) filepathContent(filepath string) (string, error) {
	if erow.IsDir() {
		// TODO: context to allow stop
		return cmdutil.ListDir(filepath, false, true)
	}
	// file content
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		return "", err
	}
	return string(b), nil
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

	erow.state.savedHash.size = len(str)
	erow.state.savedHash.hash = erow.contentHash([]byte(str))

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

func (erow *ERow) TextAreaAppendAsync(str string) <-chan struct{} {
	comm := make(chan struct{}, 1)
	erow.ed.ui.RunOnUIGoRoutine(func() {
		erow.textAreaAppend(str)
		comm <- struct{}{}
	})
	return comm
}

func (erow *ERow) textAreaAppend(str string) {
	// TODO: unlimited output? some xterms have more or less 64k limit. Bigger limits will slow down the ui since it will be calculating the new string content. This will be improved once the textarea drawer supports append/cutTop operations.

	ta := erow.row.TextArea

	maxSize := 64 * 1024
	str2 := ta.Str() + str
	if len(str2) > maxSize {
		d := len(str2) - maxSize
		str2 = str2[d:]
	}

	// false,true = keep pos, but clear undo for massive savings
	ta.SetStrClear(str2, false, true)
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
		erow.updateDuplicatesHighlight(true)
	case *event.MouseLeave:
		erow.updateDuplicatesHighlight(false)
	}
}

func (erow *ERow) updateDuplicatesHighlight(v bool) {
	erows := erow.Duplicates()
	if !v || (v && len(erows) >= 2) {
		for _, e := range erows {
			e.Row().SetState(ui.DuplicateHighlightRowState, v)
		}
	}
}

func (erow *ERow) UpdateStateAndDuplicates() {
	erow.updateStateAndDuplicates(false)
}
func (erow *ERow) updateStateAndDuplicates(isNew bool) {
	erow.updateState()
	erow.UpdateStateUI()

	// update duplicates
	if !isNew {
		otherDups := erow.OtherDuplicates()
		for _, e := range otherDups {
			if e == erow {
				continue
			}
			// update duplicate content before updating state
			erow.updateDuplicateContent(e)

			// update duplicate state
			e.state = erow.state
			e.UpdateStateUI()
		}
	}

	// update duplicates highlighting
	if isNew {
		p, err := erow.Ed().UI().QueryPointer()
		if err == nil {
			dups := erow.Duplicates()
			if len(dups) >= 2 {
				for _, e := range dups {
					if p.In(e.row.Bounds) {
						erow.updateDuplicatesHighlight(true)
						break
					}
				}
			}
		}
	}
}

func (erow *ERow) updateDuplicateContent(dstERow *ERow) {
	srcERow := erow

	// don't update itself - it will erase its own history
	if dstERow == srcERow {
		return
	}

	// update duplicate content only for files
	if !dstERow.IsRegular() {
		return
	}

	srcTa := srcERow.Row().TextArea
	dstTa := dstERow.Row().TextArea

	// keep data for later restoration
	oy := dstTa.OffsetY()
	ip := dstTa.GetPoint(dstTa.CursorIndex())

	// tmp history to set the string in the duplicate
	dstTa.History = tahistory.NewHistory(1)

	// Annotations (share instance).
	dstTa.Drawer.Args.AnnotationsOpt = srcTa.Drawer.Args.AnnotationsOpt
	// calc due to annotations - if the str content is the same the calc won't be triggered below when setting the string, need to do it here
	if srcTa.Str() == dstTa.Str() {
		dstTa.CalcChildsBounds()
	}

	dstERow.disableTextAreaSetStrEventHandler = true // avoid recursive events
	dstTa.SetStrClear(srcTa.Str(), false, false)
	dstERow.disableTextAreaSetStrEventHandler = false

	// discard tmp history and use src history to allow undo/redo (share instance)
	dstTa.History = srcTa.History

	// restore position (avoid cursor moving while editing in another row)
	dstTa.SetOffsetY(oy)
	dstTa.SetCursorIndex(dstTa.GetIndex(&ip))
}

func (erow *ERow) Duplicates() []*ERow {
	return erow.ed.FindERows(erow.Filename())
}
func (erow *ERow) OtherDuplicates() []*ERow {
	u := []*ERow{}
	for _, e := range erow.Duplicates() {
		if erow == e {
			continue
		}
		u = append(u, e)
	}
	return u
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
	ta := erow.row.TextArea
	switch filepath.Ext(erow.Filename()) {
	default:
		fallthrough
	case "", ".sh", ".conf", ".list", ".txt":
		ta.SetCommentStrings("#", [2]string{})
	case ".go", ".c", ".cpp", ".h", ".hpp":
		ta.SetCommentStrings("//", [2]string{"/*", "*/"})
	}
}

func (erow *ERow) StartExecState() context.Context {
	erow.execState.Lock()
	defer erow.execState.Unlock()

	// clear old context if exists
	if erow.execState.ctx != nil {
		erow.clearExecState2(erow.execState.ctx, nil)
	}

	// indicate the row is running
	erow.Ed().UI().RunOnUIGoRoutine(func() {
		erow.row.SetState(ui.ExecutingRowState, true)
	})

	// new context
	erow.execState.ctx, erow.execState.cancel = context.WithCancel(context.Background())
	return erow.execState.ctx
}
func (erow *ERow) StopExecState() {
	erow.execState.Lock()
	defer erow.execState.Unlock()
	if erow.execState.cancel != nil {
		erow.execState.cancel()
	}
}
func (erow *ERow) ClearExecState(ctx context.Context, fn func()) {
	erow.execState.Lock()
	defer erow.execState.Unlock()
	erow.clearExecState2(ctx, fn)
}
func (erow *ERow) clearExecState2(ctx context.Context, fn func()) {
	stop := false

	if ctx == nil {
		// stop if arg is nil (stop current context)
		stop = true
	} else {
		// stop the requested context (other context could be running already)
		if erow.execState.ctx == ctx {
			stop = true
		}
	}

	if stop {
		// run function since still running in the requested context
		if fn != nil {
			fn()
		}

		erow.execState.cancel() // clear resources
		erow.execState.ctx = nil
		erow.execState.cancel = nil

		// indicate the row is not running
		erow.Ed().UI().RunOnUIGoRoutine(func() {
			erow.row.SetState(ui.ExecutingRowState, false)
		})
	}
}

// Caller is responsible for closing the writer at the end.
func (erow *ERow) TextAreaWriter() io.WriteCloser {
	pr, pw := io.Pipe()
	go func() {
		erow.readLoopToTextArea(pr)
	}()
	return pw
}

func (erow *ERow) readLoopToTextArea(reader io.Reader) {
	var buf [32 * 1024]byte
	for {
		n, err := reader.Read(buf[:])
		if n > 0 {
			str := string(buf[:n])
			c := erow.TextAreaAppendAsync(str)

			// Wait for the ui to have handled the content. This prevents a tight loop program from leaving the UI unresponsive.
			<-c
		}
		if err != nil {
			break
		}
	}
}
