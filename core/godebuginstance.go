package core

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jmigpin/editor/core/godebug"
	"github.com/jmigpin/editor/core/godebug/debug"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/ctxutil"
	"github.com/jmigpin/editor/util/drawutil/drawer4"
	"github.com/jmigpin/editor/util/parseutil"
)

//godebug:annotatefile

// Note: Should have a unique instance because there is no easy solution to debug two (or more) programs that have common files in the same editor

const updatesPerSecond = 15

type GoDebugManager struct {
	ed   *Editor
	inst struct {
		sync.Mutex
		inst   *GoDebugInstance
		cancel context.CancelFunc
	}
}

func NewGoDebugManager(ed *Editor) *GoDebugManager {
	gdm := &GoDebugManager{ed: ed}
	return gdm
}

func (gdm *GoDebugManager) Printf(format string, args ...interface{}) {
	gdm.ed.Messagef("godebug: "+format, args...)
}

func (gdm *GoDebugManager) RunAsync(reqCtx context.Context, erow *ERow, args []string) error {

	gdm.inst.Lock()
	defer gdm.inst.Unlock()

	// cancel and wait for previous
	gdm.cancelAndWait()

	// setup instance context // TODO: editor ctx?
	ctx, cancel := context.WithCancel(context.Background())

	// call cancel if reqCtx is done (short amount of time, just watches start)
	clearWatching := ctxutil.WatchDone(cancel, reqCtx)
	defer clearWatching()

	inst, err := startGoDebugInstance(ctx, gdm.ed, gdm, erow, args)
	if err != nil {
		cancel()
		return err
	}

	gdm.inst.inst = inst
	gdm.inst.cancel = cancel
	return nil
}

func (gdm *GoDebugManager) CancelAndClear() {
	gdm.inst.Lock()
	defer gdm.inst.Unlock()
	gdm.cancelAndWait() // clears
}
func (gdm *GoDebugManager) cancelAndWait() {
	if gdm.inst.inst != nil {
		gdm.inst.cancel()
		gdm.inst.inst.wait()
		gdm.inst.inst = nil
		gdm.inst.cancel = nil
	}
}

func (gdm *GoDebugManager) SelectAnnotation(rowPos *ui.RowPos, ev *ui.RootSelectAnnotationEvent) {
	gdm.inst.Lock()
	defer gdm.inst.Unlock()
	if gdm.inst.inst != nil {
		gdm.inst.inst.selectAnnotation(rowPos, ev)
	}
}

func (gdm *GoDebugManager) SelectERowAnnotation(erow *ERow, ev *ui.TextAreaSelectAnnotationEvent) {
	gdm.inst.Lock()
	defer gdm.inst.Unlock()
	if gdm.inst.inst != nil {
		gdm.inst.inst.selectERowAnnotation(erow, ev)
	}
}

func (gdm *GoDebugManager) AnnotationFind(s string) error {
	gdm.inst.Lock()
	defer gdm.inst.Unlock()
	if gdm.inst.inst == nil {
		return fmt.Errorf("missing godebug instance")
	}
	return gdm.inst.inst.annotationFind(s)
}

func (gdm *GoDebugManager) UpdateUIERowInfo(info *ERowInfo) {
	gdm.inst.Lock()
	defer gdm.inst.Unlock()
	if gdm.inst.inst != nil {
		gdm.inst.inst.updateUIERowInfo(info)
	}
}

//----------

type GoDebugInstance struct {
	ed           *Editor
	gdm          *GoDebugManager
	di           *GDDataIndex
	erowExecWait sync.WaitGroup
}

func startGoDebugInstance(ctx context.Context, ed *Editor, gdm *GoDebugManager, erow *ERow, args []string) (*GoDebugInstance, error) {
	gdi := &GoDebugInstance{ed: ed, gdm: gdm}
	gdi.di = NewGDDataIndex(ed)
	if err := gdi.start2(ctx, erow, args); err != nil {
		return nil, err
	}
	return gdi, nil
}

func (gdi *GoDebugInstance) wait() {
	gdi.erowExecWait.Wait()
	gdi.clearInfosUI()
}

//----------

func (gdi *GoDebugInstance) start2(ctx context.Context, erow *ERow, args []string) error {
	// warn other annotators about starting a godebug session
	_ = gdi.ed.CanModifyAnnotations(EdAnnReqGoDebug, erow.Row.TextArea, "starting_session")

	// create new erow if necessary
	if erow.Info.IsFileButNotDir() {
		dir := filepath.Dir(erow.Info.Name())
		info := erow.Ed.ReadERowInfo(dir)
		rowPos := erow.Row.PosBelow()
		erow = NewBasicERow(info, rowPos)
	}

	if !erow.Info.IsDir() {
		return fmt.Errorf("can't run on this erow type")
	}

	gdi.erowExecWait.Add(1)
	erow.Exec.RunAsync(func(erowCtx context.Context, rw io.ReadWriter) error {
		defer gdi.erowExecWait.Done()

		// call cancel if ctx is done (allow cancel from godebugmanager)
		erowCtx2, cancel := context.WithCancel(erowCtx)
		defer cancel()
		clearWatching := ctxutil.WatchDone(cancel, ctx)
		defer clearWatching()

		return gdi.runCmd(erowCtx2, erow, args, rw)
	})

	return nil
}

func (gdi *GoDebugInstance) runCmd(ctx context.Context, erow *ERow, args []string, w io.Writer) error {
	cmd := godebug.NewCmd()
	defer cmd.Cleanup()

	cmd.Dir = erow.Info.Name()
	cmd.Stdout = w
	cmd.Stderr = w

	done, err := cmd.Start(ctx, args[1:])
	if err != nil {
		return err
	}
	if done {
		return nil
	}

	gdi.clientMsgsLoop(ctx, w, cmd) // blocking

	return cmd.Wait()
}

//----------

func (gdi *GoDebugInstance) selectERowAnnotation(erow *ERow, ev *ui.TextAreaSelectAnnotationEvent) {
	if gdi.selectERowAnnotation2(erow, ev) {
		gdi.updateUIShowLine(erow.Row.PosBelow())
	}
}

func (gdi *GoDebugInstance) selectERowAnnotation2(erow *ERow, ev *ui.TextAreaSelectAnnotationEvent) bool {
	switch ev.Type {
	case ui.TASelAnnTypeCurrent,
		ui.TASelAnnTypeCurrentPrev,
		ui.TASelAnnTypeCurrentNext:
		return gdi.di.annMsgChangeCurrent(erow.Info.Name(), ev.AnnotationIndex, ev.Type)
	case ui.TASelAnnTypePrint:
		gdi.printIndex(erow, ev.AnnotationIndex, ev.Offset)
		return false
	case ui.TASelAnnTypePrintAllPrevious:
		gdi.printIndexAllPrevious(erow, ev.AnnotationIndex, ev.Offset)
		return false
	default:
		log.Printf("todo: %#v", ev)
	}
	return false
}

//----------

func (gdi *GoDebugInstance) annotationFind(s string) error {
	_, ok := gdi.di.selectedAnnFind(s)
	if !ok {
		return fmt.Errorf("string not found in selected annotation: %v", s)
	}
	gdi.updateUIShowLine(gdi.ed.GoodRowPos())
	return nil
}

//----------

func (gdi *GoDebugInstance) selectAnnotation(rowPos *ui.RowPos, ev *ui.RootSelectAnnotationEvent) {
	if gdi.selectAnnotation2(ev) {
		gdi.updateUIShowLine(rowPos)
	}
}

func (gdi *GoDebugInstance) selectAnnotation2(ev *ui.RootSelectAnnotationEvent) bool {
	switch ev.Type {
	case ui.RootSelAnnTypeFirst:
		_ = gdi.di.selectFirst()
		gdi.openArrivalIndexERow()
		return true // show always
	case ui.RootSelAnnTypeLast:
		_ = gdi.di.selectLast()
		gdi.openArrivalIndexERow()
		return true // show always
	case ui.RootSelAnnTypePrev:
		_ = gdi.di.selectPrev()
		gdi.openArrivalIndexERow()
		return true // show always
	case ui.RootSelAnnTypeNext:
		_ = gdi.di.selectNext()
		gdi.openArrivalIndexERow()
		return true // show always
	case ui.RootSelAnnTypeClear:
		gdi.di.clearMsgs()
		return true
	default:
		log.Printf("todo: %#v", ev)
	}
	return false
}

//----------

func (gdi *GoDebugInstance) printIndex(erow *ERow, annIndex, offset int) {
	msg, ok := gdi.di.annMsg(erow.Info.Name(), annIndex)
	if !ok {
		return
	}
	// build output
	s := godebug.StringifyItemFull(msg.dbgLineMsg.Item)
	gdi.gdm.Printf("annotation: #%d\n\t%v\n", msg.arrivalIndex, s)
}

func (gdi *GoDebugInstance) printIndexAllPrevious(erow *ERow, annIndex, offset int) {
	msgs, ok := gdi.di.annPreviousMsgs(erow.Info.Name(), annIndex)
	if !ok {
		return
	}
	// build output
	sb := strings.Builder{}
	for _, msg := range msgs {
		s := godebug.StringifyItemFull(msg.dbgLineMsg.Item)
		sb.WriteString(fmt.Sprintf("\t" + s + "\n"))
	}
	gdi.gdm.Printf("annotations (%d entries):\n%v\n", len(msgs), sb.String())
}

//----------

func (gdi *GoDebugInstance) clientMsgsLoop(ctx context.Context, w io.Writer, cmd *godebug.Cmd) {
	var updatec <-chan time.Time // update channel
	updateUI := func() {
		if updatec != nil {
			updatec = nil
			gdi.updateUI()
		}
	}

	for {
		select {
		case <-ctx.Done():
			updateUI() // final ui update
			return
		case msg, ok := <-cmd.Client.Messages:
			if !ok {
				updateUI() // last msg (end of program), final ui update
				return
			}
			if err := gdi.handleMsg(msg, cmd); err != nil {
				fmt.Fprintf(w, "error: %v\n", err)
			}
			if updatec == nil {
				t := time.NewTimer(time.Second / updatesPerSecond)
				updatec = t.C
			}
		case <-updatec:
			updateUI()
		}
	}
}

//----------

func (gdi *GoDebugInstance) handleMsg(msg interface{}, cmd *godebug.Cmd) error {
	switch t := msg.(type) {
	case error:
		return t
	case string:
		if t == "connected" {
			// TODO: timeout to receive filesetpositions?
			// request file positions
			if err := cmd.RequestFileSetPositions(); err != nil {
				return fmt.Errorf("request file set positions: %w", err)
			}
		} else {
			return fmt.Errorf("unhandled string: %v", t)
		}
	case *debug.FilesDataMsg:
		if err := gdi.di.handleFilesDataMsg(t); err != nil {
			return err
		}
		// on receiving the filesdatamsg, send a requeststart
		if err := cmd.RequestStart(); err != nil {
			return fmt.Errorf("request start: %w", err)
		}
	case *debug.LineMsg:
		return gdi.di.handleLineMsgs(t)
	case []*debug.LineMsg:
		return gdi.di.handleLineMsgs(t...)
	default:
		return fmt.Errorf("unexpected msg: %T", msg)
	}
	return nil
}

//----------

func (gdi *GoDebugInstance) updateUI() {
	gdi.ed.UI.RunOnUIGoRoutine(func() {
		gdi.updateUI2()
	})
}

func (gdi *GoDebugInstance) updateUIShowLine(rowPos *ui.RowPos) {
	gdi.ed.UI.RunOnUIGoRoutine(func() {
		gdi.updateUI2()
		gdi.showSelectedLine(rowPos)
	})
}

func (gdi *GoDebugInstance) updateUIERowInfo(info *ERowInfo) {
	gdi.ed.UI.RunOnUIGoRoutine(func() {
		gdi.updateInfoUI(info)
	})
}

//----------

func (gdi *GoDebugInstance) clearInfosUI() {
	gdi.ed.UI.RunOnUIGoRoutine(func() {
		for _, info := range gdi.ed.ERowInfos() {
			gdi.clearInfoUI(info)
		}
	})
}

func (gdi *GoDebugInstance) clearInfoUI(info *ERowInfo) {
	info.UpdateAnnotationsRowState(false)
	info.UpdateAnnotationsEditedRowState(false)
	gdi.clearAnnotations(info)
}

//----------

func (gdi *GoDebugInstance) updateUI2() {
	for _, info := range gdi.ed.ERowInfos() {
		gdi.updateInfoUI(info)
	}
}

func (gdi *GoDebugInstance) updateInfoUI(info *ERowInfo) {
	// TODO: the info should be get with one locked call to the dataindex

	// Note: the current selected debug line might not have an open erow (ex: when auto increased to match the lastarrivalindex).

	// file belongs to the godebug session
	findex, ok := gdi.di.FilesIndex(info.Name())
	if !ok {
		info.UpdateAnnotationsRowState(false)
		info.UpdateAnnotationsEditedRowState(false)
		gdi.clearAnnotations(info)
		return
	}
	info.UpdateAnnotationsRowState(true)

	// check if content has changed
	edited := gdi.di.updateFileEdited(info)
	if edited {
		info.UpdateAnnotationsEditedRowState(true)
		gdi.clearAnnotations(info)
		return
	}
	info.UpdateAnnotationsEditedRowState(false)

	selLine, ok := gdi.di.findSelectedAndUpdateAnnEntries(findex)
	if !ok {
		selLine = -1
	}

	// set annotations
	file := gdi.di.files[findex] // TODO: not locked (file.AnnEntries used)
	for _, erow := range info.ERows {
		gdi.setAnnotations(erow, true, selLine, file.annEntries)
	}
}

func (gdi *GoDebugInstance) clearAnnotations(info *ERowInfo) {
	for _, erow := range info.ERows {
		gdi.setAnnotations(erow, false, -1, nil)
	}
}

func (gdi *GoDebugInstance) setAnnotations(erow *ERow, on bool, selIndex int, entries []*drawer4.Annotation) {
	gdi.ed.SetAnnotations(EdAnnReqGoDebug, erow.Row.TextArea, on, selIndex, entries)
}

//----------

func (gdi *GoDebugInstance) showSelectedLine(rowPos *ui.RowPos) {
	msg, filename, arrivalIndex, edited, ok := gdi.di.selectedMsg()
	if !ok {
		return
	}

	// TODO: don't show if on UI list, show warnings about skipped steps
	// some rows show because the selected arrival index is just increased
	// but in the case of searching for the next selected arrival index, if the info row is not opened, it doesn't search inside that file, and so the index stays the same as the last selected index

	// don't show on edited files
	if edited {
		gdi.ed.Errorf("selection at edited row: %v: step %v", filename, arrivalIndex)
		return
	}

	// file offset
	dlm := msg.dbgLineMsg
	fo := &parseutil.FilePos{Filename: filename, Offset: dlm.Offset}

	// show line
	conf := &OpenFileERowConfig{
		FilePos:             fo,
		RowPos:              rowPos,
		FlashVisibleOffsets: true,
		NewIfNotExistent:    true,
	}
	OpenFileERow(gdi.ed, conf)
}

//----------

func (gdi *GoDebugInstance) openArrivalIndexERow() {
	_, filename, ok := gdi.di.selectedArrivalIndexFilename()
	if !ok {
		return
	}

	rowPos := gdi.ed.GoodRowPos()
	conf := &OpenFileERowConfig{
		FilePos:          &parseutil.FilePos{Filename: filename},
		RowPos:           rowPos,
		CancelIfExistent: true,
		NewIfNotExistent: true,
	}
	gdi.ed.UI.RunOnUIGoRoutine(func() {
		OpenFileERow(gdi.ed, conf)
	})
}

//----------

// GoDebug data Index
type GDDataIndex struct {
	sync.RWMutex // used internally, not to be locked outside

	ed          *Editor
	filesIndexM map[string]int // [name]fileindex
	filesEdited map[int]bool   // [fileindex]

	afds  []*debug.AnnotatorFileData // [fileindex]
	files []*GDFileMsgs              // [fileindex]

	lastArrivalIndex int
	selected         struct {
		arrivalIndex  int
		fileIndex     int
		lineIndex     int
		lineStepIndex int
	}
}

func NewGDDataIndex(ed *Editor) *GDDataIndex {
	di := &GDDataIndex{ed: ed}
	di.filesIndexM = map[string]int{}
	di.filesEdited = map[int]bool{}
	di.clearMsgs()
	return di
}

func (di *GDDataIndex) FilesIndex(name string) (int, bool) {
	name = di.FilesIndexKey(name)
	v, ok := di.filesIndexM[name]
	return v, ok
}
func (di *GDDataIndex) FilesIndexKey(name string) string {
	if di.ed.FsCaseInsensitive {
		name = strings.ToLower(name)
	}
	return name
}

func (di *GDDataIndex) clearMsgs() {
	di.Lock()
	defer di.Unlock()
	for _, f := range di.files {
		n := len(f.linesMsgs) // keep n
		u := NewGDFileMsgs(n)
		*f = *u
	}
	di.lastArrivalIndex = -1
	di.selected.arrivalIndex = di.lastArrivalIndex
}

//----------

func (di *GDDataIndex) handleFilesDataMsg(fdm *debug.FilesDataMsg) error {
	di.Lock()
	defer di.Unlock()

	di.afds = fdm.Data
	// index filenames
	di.filesIndexM = map[string]int{}
	for _, afd := range di.afds {
		name := di.FilesIndexKey(afd.Filename)
		di.filesIndexM[name] = afd.FileIndex
	}
	// init index
	di.files = make([]*GDFileMsgs, len(di.afds))
	for _, afd := range di.afds {
		// check index
		if afd.FileIndex >= len(di.files) {
			return fmt.Errorf("bad file index at init: %v len=%v", afd.FileIndex, len(di.files))
		}
		di.files[afd.FileIndex] = NewGDFileMsgs(afd.DebugLen)
	}
	return nil
}

func (di *GDDataIndex) handleLineMsgs(msgs ...*debug.LineMsg) error {
	di.Lock()
	defer di.Unlock()
	for _, msg := range msgs {
		err := di._handleLineMsg(msg)
		if err != nil {
			return err
		}
	}
	return nil
}

// Not locked
func (di *GDDataIndex) _handleLineMsg(u *debug.LineMsg) error {
	// check index
	l1 := len(di.files)
	if u.FileIndex >= l1 {
		return fmt.Errorf("bad file index: %v len=%v", u.FileIndex, l1)
	}
	// check index
	l2 := len(di.files[u.FileIndex].linesMsgs)
	if u.DebugIndex >= l2 {
		return fmt.Errorf("bad debug index: %v len=%v", u.DebugIndex, l2)
	}
	// line msg
	di.lastArrivalIndex++ // starts/clears to -1, so first n is 0
	lm := &GDLineMsg{arrivalIndex: di.lastArrivalIndex, dbgLineMsg: u}
	// index msg
	w := &di.files[u.FileIndex].linesMsgs[u.DebugIndex].lineMsgs
	*w = append(*w, lm)
	// mark file as having new data (performance)
	//di.files[u.FileIndex].hasNewData = true

	// auto update selected index if at last position
	if di.selected.arrivalIndex == di.lastArrivalIndex-1 {
		di.selected.arrivalIndex = di.lastArrivalIndex
	}

	return nil
}

//----------

func (di *GDDataIndex) annMsg(filename string, annIndex int) (*GDLineMsg, bool) {
	di.RLock()
	defer di.RUnlock()

	file, line, ok := di._annIndexFileLine(filename, annIndex)
	if !ok {
		return nil, false
	}
	// current msg index at line
	k := file.annEntriesLMIndex[annIndex] // same length as lineMsgs
	if k < 0 || k >= len(line.lineMsgs) { // currently nothing is shown or cleared
		return nil, false
	}
	return line.lineMsgs[k], true
}

func (di *GDDataIndex) annPreviousMsgs(filename string, annIndex int) ([]*GDLineMsg, bool) {
	di.RLock()
	defer di.RUnlock()

	file, line, ok := di._annIndexFileLine(filename, annIndex)
	if !ok {
		return nil, false
	}
	// current msg index at line
	k := file.annEntriesLMIndex[annIndex] // same length as lineMsgs
	if k < 0 || k >= len(line.lineMsgs) { // currently nothing is shown or cleared
		return nil, false
	}
	return line.lineMsgs[:k+1], true
}

func (di *GDDataIndex) selectedAnnFind(s string) (*GDLineMsg, bool) {
	di.RLock()
	defer di.RUnlock()

	annIndex, filename, ok := di.selectedArrivalIndexFilename()
	if !ok {
		return nil, false
	}

	file, line, ok := di._annIndexFileLine(filename, annIndex)
	if !ok {
		return nil, false
	}

	b := []byte(s)
	k := file.annEntriesLMIndex[annIndex] // current entry
	for i := 0; i < len(line.lineMsgs); i++ {
		h := (k + 1 + i) % len(line.lineMsgs)
		msg := line.lineMsgs[h]
		ann := msg.annotation()
		j := bytes.Index(ann.Bytes, b)
		if j >= 0 {
			di.selected.arrivalIndex = msg.arrivalIndex
			return msg, true
		}
	}

	return nil, false
}

func (di *GDDataIndex) annMsgChangeCurrent(filename string, annIndex int, typ ui.TASelAnnType) bool {
	di.Lock() // writes di.selected
	defer di.Unlock()

	file, line, ok := di._annIndexFileLine(filename, annIndex)
	if !ok {
		return false
	}
	// current msg index at line
	k := file.annEntriesLMIndex[annIndex] // same length as lineMsgs

	// adjust k according to type
	switch typ {
	case ui.TASelAnnTypeCurrent:
		// allow to select first if no line is visible
		if k < 0 {
			k = 0
		}
	case ui.TASelAnnTypeCurrentPrev:
		k--
	case ui.TASelAnnTypeCurrentNext:
		k++
	default:
		panic(fmt.Sprintf("unexpected type: %v", typ))
	}

	if k < 0 || k >= len(line.lineMsgs) { // currently nothing is shown or cleared
		return false
	}
	di.selected.arrivalIndex = line.lineMsgs[k].arrivalIndex
	return true
}

// Not locked
func (di *GDDataIndex) _annIndexFileLine(filename string, annIndex int) (*GDFileMsgs, *GDLineMsgs, bool) {
	// file
	findex, ok := di.FilesIndex(filename)
	if !ok {
		return nil, nil, false
	}
	file := di.files[findex]
	// line
	if annIndex < 0 || annIndex >= len(file.linesMsgs) {
		return nil, nil, false
	}
	return file, &file.linesMsgs[annIndex], true
}

//----------

func (di *GDDataIndex) selectFirst() bool {
	di.Lock()
	defer di.Unlock()
	if di.selected.arrivalIndex != 0 && 0 <= di.lastArrivalIndex { // could be -1
		di.selected.arrivalIndex = 0
		return true
	}
	return false
}

func (di *GDDataIndex) selectLast() bool {
	di.Lock()
	defer di.Unlock()
	if di.selected.arrivalIndex != di.lastArrivalIndex {
		di.selected.arrivalIndex = di.lastArrivalIndex
		return true
	}
	return false
}

func (di *GDDataIndex) selectPrev() bool {
	di.Lock()
	defer di.Unlock()
	if di.selected.arrivalIndex > 0 {
		di.selected.arrivalIndex--
		return true
	}
	return false
}

func (di *GDDataIndex) selectNext() bool {
	di.Lock()
	defer di.Unlock()
	if di.selected.arrivalIndex < di.lastArrivalIndex {
		di.selected.arrivalIndex++
		return true
	}
	return false
}

//----------

func (di *GDDataIndex) findSelectedAndUpdateAnnEntries(findex int) (int, bool) {
	di.Lock()
	defer di.Unlock()
	file := di.files[findex]
	selLine, selLineStep, selFound := file._findSelectedAndUpdateAnnEntries(di.selected.arrivalIndex)
	if selFound {
		di.selected.fileIndex = findex
		di.selected.lineIndex = selLine
		di.selected.lineStepIndex = selLineStep
	}
	return selLine, selFound
}

//----------

func (di *GDDataIndex) selectedMsg() (*GDLineMsg, string, int, bool, bool) {
	di.RLock()
	defer di.RUnlock()

	msg, ok := di._selectedMsg2()
	if !ok {
		return nil, "", 0, false, false
	}

	findex := di.selected.fileIndex
	filename := di.afds[findex].Filename
	edited := di.filesEdited[findex]
	return msg, filename, di.selected.arrivalIndex, edited, true
}

// Not locked.
func (di *GDDataIndex) _selectedMsg2() (*GDLineMsg, bool) {
	// in case of a clear
	if di.selected.arrivalIndex < 0 {
		return nil, false
	}

	findex := di.selected.fileIndex
	if findex < 0 || findex >= len(di.files) {
		return nil, false
	}
	file := di.files[findex]

	lineIndex := di.selected.lineIndex
	if lineIndex < 0 || lineIndex >= len(file.linesMsgs) {
		return nil, false
	}
	lm := file.linesMsgs[lineIndex]

	stepIndex := di.selected.lineStepIndex
	if stepIndex < 0 || stepIndex >= len(lm.lineMsgs) {
		return nil, false
	}

	return lm.lineMsgs[stepIndex], true
}

//----------

func (di *GDDataIndex) updateFileEdited(info *ERowInfo) bool {
	di.Lock()
	defer di.Unlock()
	findex, ok := di.FilesIndex(info.Name())
	if !ok {
		return false
	}
	afd := di.afds[findex]
	edited := !info.EqualToBytesHash(afd.FileSize, afd.FileHash)
	di.filesEdited[findex] = edited
	return edited
}

func (di *GDDataIndex) isFileEdited(filename string) bool {
	di.RLock()
	defer di.RUnlock()
	findex, ok := di.FilesIndex(filename)
	if !ok {
		return false
	}
	return di.filesEdited[findex]
}

//----------

func (di *GDDataIndex) selectedArrivalIndexFilename() (int, string, bool) {
	return di.arrivalIndexFilename(di.selected.arrivalIndex)
}

func (di *GDDataIndex) arrivalIndexFilename(arrivalIndex int) (int, string, bool) {
	di.RLock()
	defer di.RUnlock()
	for findex, file := range di.files {
		for j, lm := range file.linesMsgs {
			_, eqK, _ := lm.findIndex(arrivalIndex)
			if eqK {
				return j, di.afds[findex].Filename, true
			}
		}
	}
	return -1, "", false
}

//----------

type GDFileMsgs struct {
	linesMsgs []GDLineMsgs // [lineIndex] file annotations received

	// current annotation entries to be shown with a file
	annEntries        []*drawer4.Annotation
	annEntriesLMIndex []int // [lineIndex]stepIndex: line messages index: keep selected step index to know the msg entry when coming from a click on an annotation

	//hasNewData bool // performance
}

func NewGDFileMsgs(n int) *GDFileMsgs {
	return &GDFileMsgs{
		linesMsgs:         make([]GDLineMsgs, n),
		annEntries:        make([]*drawer4.Annotation, n),
		annEntriesLMIndex: make([]int, n),
	}
}

// Not locked
func (file *GDFileMsgs) _findSelectedAndUpdateAnnEntries(arrivalIndex int) (int, int, bool) {
	found := false
	selLine := 0
	selLineStep := 0
	for line, lm := range file.linesMsgs {
		k, eqK, foundK := lm.findIndex(arrivalIndex)
		if foundK {
			file.annEntries[line] = lm.lineMsgs[k].annotation()
			file.annEntriesLMIndex[line] = k
			if eqK {
				found = true
				selLine = line
				selLineStep = k
			}
		} else {
			if len(lm.lineMsgs) > 0 {
				file.annEntries[line] = lm.lineMsgs[0].emptyAnnotation()
			} else {
				file.annEntries[line] = nil // no msgs ever received
			}
			file.annEntriesLMIndex[line] = -1
		}
	}
	return selLine, selLineStep, found
}

//----------

type GDLineMsgs struct {
	lineMsgs []*GDLineMsg // [arrivalIndex] line annotations received
}

func (lm *GDLineMsgs) findIndex(arrivalIndex int) (int, bool, bool) {
	k := sort.Search(len(lm.lineMsgs), func(i int) bool {
		u := lm.lineMsgs[i].arrivalIndex
		return u >= arrivalIndex
	})
	foundK := false
	eqK := false
	if k < len(lm.lineMsgs) && lm.lineMsgs[k].arrivalIndex == arrivalIndex {
		eqK = true
		foundK = true
	} else {
		k-- // current k is above arrivalIndex, want previous
		foundK = k >= 0
	}
	return k, eqK, foundK
}

//----------

type GDLineMsg struct {
	arrivalIndex int
	dbgLineMsg   *debug.LineMsg
	cache        struct {
		item []byte
		ann  *drawer4.Annotation
	}
}

func (msg *GDLineMsg) ann() *drawer4.Annotation {
	if msg.cache.ann == nil {
		msg.cache.ann = &drawer4.Annotation{Offset: msg.dbgLineMsg.Offset}
	}
	return msg.cache.ann
}

func (msg *GDLineMsg) annotation() *drawer4.Annotation {
	ann := msg.ann()
	if msg.cache.item == nil {
		s := godebug.StringifyItem(msg.dbgLineMsg.Item)
		msg.cache.item = []byte(s)
	}
	ann.Bytes = msg.cache.item
	ann.NotesBytes = []byte(fmt.Sprintf("#%d", msg.arrivalIndex))
	return ann
}

func (msg *GDLineMsg) emptyAnnotation() *drawer4.Annotation {
	ann := msg.ann()
	ann.Bytes = []byte(" ")
	ann.NotesBytes = nil
	return ann
}
