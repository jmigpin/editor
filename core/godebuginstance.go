package core

import (
	"bytes"
	"context"
	"fmt"
	"io"
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

// Note: Should have a unique instance because there is no easy solution to debug two (or more) programs that have common files in the same editor

const updatesPerSecond = 12

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

func (gdm *GoDebugManager) UpdateInfoAnnotations(info *ERowInfo) {
	gdm.inst.Lock()
	defer gdm.inst.Unlock()
	if gdm.inst.inst != nil {
		gdm.inst.inst.updateInfoAnnotations(info)
	}
}

//----------

func (gdm *GoDebugManager) Printf(format string, args ...any) {
	gdm.ed.Messagef("godebug: "+format, args...)
}
func (gdm *GoDebugManager) printError(err error) {
	gdm.ed.Errorf("godebug: %w", err)
}

//----------
//----------
//----------

type GoDebugInstance struct {
	gdm          *GoDebugManager
	di           *GDDataIndex
	erowExecWait sync.WaitGroup
}

func startGoDebugInstance(ctx context.Context, ed *Editor, gdm *GoDebugManager, erow *ERow, args []string) (*GoDebugInstance, error) {
	gdi := &GoDebugInstance{gdm: gdm}
	gdi.di = NewGDDataIndex(gdi)
	if err := gdi.start2(ctx, erow, args); err != nil {
		return nil, err
	}
	return gdi, nil
}

func (gdi *GoDebugInstance) wait() {
	gdi.erowExecWait.Wait()
	gdi.clearAnnotations()
}

//----------

func (gdi *GoDebugInstance) start2(ctx context.Context, erow *ERow, args []string) error {
	// warn other annotators about starting a godebug session
	_ = gdi.gdm.ed.CanModifyAnnotations(EareqGoDebugStart, erow.Row.TextArea)

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

	if err := gdi.di.handleFilesDataMsg(cmd.ProtoFilesData()); err != nil {
		return err
	}

	gdi.messagesLoop(w, cmd) // blocking

	return cmd.Wait()
}

//----------

func (gdi *GoDebugInstance) selectAnnotation(rowPos *ui.RowPos, ev *ui.RootSelectAnnotationEvent) {
	if err := gdi.selectAnnotation2(ev); err != nil {
		//gdi.gdm.printError(err)
		//gdi.updateAnnotations()
		//return
	}
	gdi.updateAnnotationsAndShowLine(nil, rowPos)
}

func (gdi *GoDebugInstance) selectAnnotation2(ev *ui.RootSelectAnnotationEvent) error {
	switch ev.Type {
	case ui.RootSelAnnTypeFirst:
		return gdi.di.selectFirst()
	case ui.RootSelAnnTypeLast:
		return gdi.di.selectLast()
	case ui.RootSelAnnTypePrev:
		return gdi.di.selectPrev()
	case ui.RootSelAnnTypeNext:
		return gdi.di.selectNext()
	case ui.RootSelAnnTypeClear:
		gdi.di.reset()
		return nil
	default:
		return fmt.Errorf("todo: %#v", ev)
	}
}

//----------

func (gdi *GoDebugInstance) selectERowAnnotation(erow *ERow, ev *ui.TextAreaSelectAnnotationEvent) {
	if err := gdi.selectERowAnnotation2(erow, ev); err != nil {
		//gdi.gdm.printError(err)
		//gdi.updateAnnotations()
		//return
	}
	gdi.updateAnnotationsAndShowLine(erow, erow.Row.PosBelow())
}
func (gdi *GoDebugInstance) selectERowAnnotation2(erow *ERow, ev *ui.TextAreaSelectAnnotationEvent) error {
	switch ev.Type {
	case ui.TASelAnnTypePrev:
		return gdi.di.selectPrev()
	case ui.TASelAnnTypeNext:
		return gdi.di.selectNext()
	case ui.TASelAnnTypeLine,
		ui.TASelAnnTypeLinePrev,
		ui.TASelAnnTypeLineNext:
		return gdi.di.selectLineAnnotation(erow.Info.Name(), ev.AnnotationIndex, ev.Type)
	case ui.TASelAnnTypePrint:
		gdi.printIndex(erow, ev.AnnotationIndex, ev.Offset)
		return nil
	case ui.TASelAnnTypePrintAllPrevious:
		gdi.printIndexAllPrevious(erow, ev.AnnotationIndex, ev.Offset)
		return nil
	default:
		return fmt.Errorf("todo: %#v", ev)
	}
}

//----------

func (gdi *GoDebugInstance) annotationFind(s string) error {
	_, ok := gdi.di.selectedAnnFind(s)
	if !ok {
		return fmt.Errorf("string not found in selected annotation: %v", s)
	}
	gdi.updateAnnotationsAndShowLine(nil, gdi.gdm.ed.GoodRowPos())
	return nil
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
		sb.WriteString("\t" + s + "\n")
	}
	gdi.gdm.Printf("annotations (%d entries):\n%v\n", len(msgs), sb.String())
}

//----------

func (gdi *GoDebugInstance) messagesLoop(w io.Writer, cmd *godebug.Cmd) {

	updateInterval := time.Second / updatesPerSecond
	var d struct {
		sync.Mutex
		updating        bool
		lastUpdateStart time.Time
	}
	updateUI := func() { // d must be locked
		d.updating = false
		gdi.updateAnnotations()
	}
	checkUI := (func())(nil)
	checkUI = func() {
		d.Lock()
		defer d.Unlock()

		if d.updating { // update will run in the future
			return
		}
		d.updating = true

		now := time.Now()
		d.lastUpdateStart = now
		deadline := d.lastUpdateStart.Add(updateInterval)

		// update now
		if now.After(deadline) {
			updateUI()
			return
		}
		// update later
		_ = time.AfterFunc(deadline.Sub(now), func() {
			d.Lock()
			defer d.Unlock()
			updateUI()
		})
	}

	//----------

	handleError := func(err error) {
		fmt.Fprintf(w, "godebuginstance: error: %v\n", err)
		//gdi.gdm.ed.Errorf("godebuginstance: %v", err)
	}

	for {
		checkUI()

		v, ok, err := cmd.ProtoRead()
		if err != nil {
			handleError(err)
			break
		}
		if !ok {
			break
		}
		if err := gdi.handleMsg(v, cmd); err != nil {
			handleError(err)
			break
		}
	}
}
func (gdi *GoDebugInstance) handleMsg(msg any, cmd *godebug.Cmd) error {
	switch t := msg.(type) {
	case *debug.LineMsg:
		return gdi.di.handleLineMsgs(t)
	case *debug.LineMsgs:
		return gdi.di.handleLineMsgs(*t...)
	default:
		return fmt.Errorf("unexpected msg: %T", msg)
	}
}

//----------

func (gdi *GoDebugInstance) updateAnnotations() {
	gdi.gdm.ed.UI.RunOnUIGoRoutine(func() {
		gdi.updateAnnotations2()
	})
}
func (gdi *GoDebugInstance) updateAnnotationsAndShowLine(preferedERow *ERow, rowPos *ui.RowPos) {
	gdi.gdm.ed.UI.RunOnUIGoRoutine(func() {
		// ensure that current arrival index line erow is open such that updateannotations can calculate the selected index, and the showselectedline will have that index to select
		gdi.openArrivalIndexERow()

		gdi.updateAnnotations2()
		gdi.showSelectedLine(preferedERow, rowPos)
	})
}

//----------

func (gdi *GoDebugInstance) updateAnnotations2() {
	for _, info := range gdi.gdm.ed.ERowInfos() {
		gdi.updateInfoAnnotations2(info)
	}
}

//----------

func (gdi *GoDebugInstance) clearAnnotations() {
	gdi.gdm.ed.UI.RunOnUIGoRoutine(func() {
		for _, info := range gdi.gdm.ed.ERowInfos() {
			info.UpdateAnnotationsRowState(false)
			info.UpdateAnnotationsEditedRowState(false)
			gdi.clearInfoAnnotations2(info)
		}
	})
}
func (gdi *GoDebugInstance) clearInfoAnnotations2(info *ERowInfo) {
	for _, erow := range info.ERows {
		gdi.setAnnotations(erow, false, -1, nil)
	}
}

//----------

func (gdi *GoDebugInstance) updateInfoAnnotations(info *ERowInfo) {
	gdi.gdm.ed.UI.RunOnUIGoRoutine(func() {
		gdi.updateInfoAnnotations2(info)
	})
}
func (gdi *GoDebugInstance) updateInfoAnnotations2(info *ERowInfo) {
	entries, selLine, edited, fileFound := gdi.di.findSelectedAndUpdateAnnEntries(info)

	info.UpdateAnnotationsRowState(fileFound)
	info.UpdateAnnotationsEditedRowState(edited)

	if !fileFound || edited {
		gdi.clearInfoAnnotations2(info)
		return
	}

	// set annotations into opened (existing) erows
	// Note: the current selected debug line might not have an open erow (ex: when auto increased to match the lastarrivalindex).
	for _, erow := range info.ERows {
		gdi.setAnnotations(erow, true, selLine, entries)
	}
}

//----------

func (gdi *GoDebugInstance) setAnnotations(erow *ERow, on bool, selIndex int, entries *drawer4.AnnotationGroup) {
	gdi.gdm.ed.SetAnnotations(EareqGoDebug, erow.Row.TextArea, on, selIndex, entries)
}

//----------

func (gdi *GoDebugInstance) showSelectedLine(preferedERow *ERow, rowPos *ui.RowPos) {
	if err := gdi.showSelectedLine2(preferedERow, rowPos); err != nil {
		gdi.gdm.printError(err)
	}
}
func (gdi *GoDebugInstance) showSelectedLine2(preferedERow *ERow, rowPos *ui.RowPos) error {
	// ensure the selected line filename has the calculation done in case the erow was not open

	msg, filename, arrivalIndex, edited, err := gdi.di.selectedMsg()
	if err != nil {
		return err
	}

	// don't show on edited files
	if edited {
		return fmt.Errorf("selection at edited row: %v: step %v", filename, arrivalIndex)
	}

	// show line // NOTE: if the row was not opened, on open, it will try to read annotation info
	conf := &OpenFileERowConfig{
		FilePos: &parseutil.FilePos{
			Filename: filename,
			Offset:   int(msg.dbgLineMsg.Offset),
		},
		RowPos:              rowPos,
		FlashVisibleOffsets: true,
		NewIfNotExistent:    true,
		PreferedERow:        preferedERow,
	}
	OpenFileERow(gdi.gdm.ed, conf)
	return nil
}

//----------

func (gdi *GoDebugInstance) openArrivalIndexERow() {
	_, filename, ok := gdi.di.selectedArrivalIndexFilename()
	if !ok {
		return
	}
	rowPos := gdi.gdm.ed.GoodRowPos()
	conf := &OpenFileERowConfig{
		FilePos:          &parseutil.FilePos{Filename: filename},
		RowPos:           rowPos,
		CancelIfExistent: true,
		NewIfNotExistent: true,
	}
	OpenFileERow(gdi.gdm.ed, conf)
}

//----------
//----------
//----------

// GoDebug data Index
type GDDataIndex struct {
	sync.RWMutex

	gdi *GoDebugInstance

	afds        []*debug.AnnotatorFileData // [fileindex]
	files       []*GDFileMsgs              // [fileindex]
	filesEdited map[int]bool               // [fileindex]
	filesIndexM map[string]int             // [name]fileindex

	lastArrivalIndex int
	selected         struct {
		arrivalIndex int

		// TODO: review, after reset(), what is the state of this
		fileIndex     int
		lineIndex     int
		lineStepIndex int
	}
}

func NewGDDataIndex(gdi *GoDebugInstance) *GDDataIndex {
	di := &GDDataIndex{gdi: gdi}
	di.filesIndexM = map[string]int{}
	di.filesEdited = map[int]bool{}
	di.reset()
	return di
}

//----------

func (di *GDDataIndex) FilesIndex(name string) (int, bool) {
	name = di.FilesIndexKey(name)
	v, ok := di.filesIndexM[name]
	return v, ok
}
func (di *GDDataIndex) FilesIndexKey(name string) string {
	if di.gdi.gdm.ed.FsCaseInsensitive {
		name = strings.ToLower(name)
	}
	return name
}

func (di *GDDataIndex) reset() {
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
		di.filesIndexM[name] = int(afd.FileIndex)
	}
	// init index
	di.files = make([]*GDFileMsgs, len(di.afds))
	for _, afd := range di.afds {
		// check index
		if int(afd.FileIndex) >= len(di.files) {
			return fmt.Errorf("bad file index at init: %v len=%v", afd.FileIndex, len(di.files))
		}
		di.files[afd.FileIndex] = NewGDFileMsgs(int(afd.DebugNIndexes))
	}
	return nil
}

func (di *GDDataIndex) handleLineMsgs(msgs ...*debug.LineMsg) error {
	di.Lock()
	defer di.Unlock()
	for _, msg := range msgs {
		err := di.handleLineMsg_noLock(msg)
		if err != nil {
			return err
		}
	}
	return nil
}
func (di *GDDataIndex) handleLineMsg_noLock(u *debug.LineMsg) error {
	// check index
	l1 := len(di.files)
	if int(u.FileIndex) >= l1 {
		return fmt.Errorf("bad file index: %v len=%v", u.FileIndex, l1)
	}
	// check index
	l2 := len(di.files[int(u.FileIndex)].linesMsgs)
	if int(u.DebugIndex) >= l2 {
		return fmt.Errorf("bad debug index: %v len=%v", u.DebugIndex, l2)
	}
	// line msg
	di.lastArrivalIndex++ // starts/clears to -1, so first n is 0
	lm := &GDLineMsg{arrivalIndex: di.lastArrivalIndex, dbgLineMsg: u}
	// append newly arrived line msg
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

	file, line, ok := di.annIndexFileLine_noLock(filename, annIndex)
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

	file, line, ok := di.annIndexFileLine_noLock(filename, annIndex)
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

	file, line, ok := di.annIndexFileLine_noLock(filename, annIndex)
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
func (di *GDDataIndex) selectLineAnnotation(filename string, annIndex int, typ ui.TASelAnnType) error {
	di.Lock() // writes di.selected
	defer di.Unlock()

	file, line, ok := di.annIndexFileLine_noLock(filename, annIndex)
	if !ok {
		return fmt.Errorf("file not indexed: %v", filename)
	}

	if len(line.lineMsgs) == 0 {
		return fmt.Errorf("no msgs in this line yet")
	}

	// current msg index at line
	k := file.annEntriesLMIndex[annIndex] // same length as lineMsgs

	// annotation already selected before attempting to change
	selected := k >= 0 && k < len(line.lineMsgs) && line.lineMsgs[k].arrivalIndex == di.selected.arrivalIndex

	// no line selected yet on this line
	if k < 0 {
		k = 0 // NOTE: there is at least one line (tested above)
	}

	// from here: k>=0

	// adjust k according to type
	switch typ {
	case ui.TASelAnnTypeLine:
		// might be selected already or not
	case ui.TASelAnnTypeLinePrev:
		if k == 0 {
			if selected {
				return fmt.Errorf("already at line first index")
			}
		} else {
			k--
		}
	case ui.TASelAnnTypeLineNext:
		if k >= len(line.lineMsgs)-1 {
			if selected {
				return fmt.Errorf("already at line last index")
			}
		} else {
			k++
		}
	default:
		panic(fmt.Sprintf("unexpected type: %v", typ))
	}

	di.selected.arrivalIndex = line.lineMsgs[k].arrivalIndex
	return nil
}
func (di *GDDataIndex) annIndexFileLine_noLock(filename string, annIndex int) (*GDFileMsgs, *GDLineMsgs, bool) {
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

	return file, file.linesMsgs[annIndex], true
}

//----------

func (di *GDDataIndex) selectFirst() error {
	di.Lock()
	defer di.Unlock()
	if di.lastArrivalIndex < 0 {
		return fmt.Errorf("no indexes arrived yet")
	}
	if di.selected.arrivalIndex == 0 {
		return fmt.Errorf("already at first index")
	}
	di.selected.arrivalIndex = 0
	return nil
}

func (di *GDDataIndex) selectLast() error {
	di.Lock()
	defer di.Unlock()
	if di.lastArrivalIndex < 0 {
		return fmt.Errorf("no indexes arrived yet")
	}
	if di.selected.arrivalIndex == di.lastArrivalIndex {
		return fmt.Errorf("already at last index")
	}
	di.selected.arrivalIndex = di.lastArrivalIndex
	return nil
}

func (di *GDDataIndex) selectPrev() error {
	di.Lock()
	defer di.Unlock()
	if di.lastArrivalIndex < 0 {
		return fmt.Errorf("no indexes arrived yet")
	}
	if di.selected.arrivalIndex == 0 {
		return fmt.Errorf("already at first index")
	}
	if di.selected.arrivalIndex < 0 { // not selected yet
		di.selected.arrivalIndex = di.lastArrivalIndex
		return nil
	}
	di.selected.arrivalIndex--
	return nil
}

func (di *GDDataIndex) selectNext() error {
	di.Lock()
	defer di.Unlock()
	if di.lastArrivalIndex < 0 {
		return fmt.Errorf("no indexes arrived yet")
	}
	if di.selected.arrivalIndex == di.lastArrivalIndex {
		return fmt.Errorf("already at last index")
	}
	if di.selected.arrivalIndex < 0 { // not selected yet
		di.selected.arrivalIndex = di.lastArrivalIndex
		return nil
	}
	di.selected.arrivalIndex++
	return nil
}

//----------

//----------

func (di *GDDataIndex) findSelectedAndUpdateAnnEntries(info *ERowInfo) (entries *drawer4.AnnotationGroup, selLine int, edited bool, fileFound bool) {
	di.Lock()
	defer di.Unlock()

	// info.name must be in the debug session
	findex, ok := di.FilesIndex(info.Name())
	if !ok {
		return nil, 0, false, false
	}

	if edited = di.updateFileEdited_noLock(info, findex); edited {
		return nil, 0, edited, true
	}

	if selLine, ok = di.findSelectedAndUpdateAnnEntries_noLock(findex); !ok {
		selLine = -1
	}
	return di.files[findex].annEntries, selLine, edited, true
}

func (di *GDDataIndex) findSelectedAndUpdateAnnEntries_noLock(findex int) (int, bool) {
	file := di.files[findex]
	selLine, selLineStep, selFound := file.findSelectedAndUpdateAnnEntries(di.selected.arrivalIndex)
	if selFound {
		di.selected.fileIndex = findex
		di.selected.lineIndex = selLine
		di.selected.lineStepIndex = selLineStep
	}
	return selLine, selFound
}

//----------

func (di *GDDataIndex) selectedMsg() (*GDLineMsg, string, int, bool, error) {
	di.RLock()
	defer di.RUnlock()

	// the di.selected.* fields are updated on updateannotations()
	// but the update annotations only updates opened info's

	// in case of a clear
	if di.selected.arrivalIndex < 0 {
		return nil, "", 0, false, fmt.Errorf("bad selected arrival index: %v", di.selected.arrivalIndex)
	}

	findex := di.selected.fileIndex
	filename := di.afds[findex].Filename
	edited := di.filesEdited[findex]

	msg, err := di.selectedMsg_noLock()
	if err != nil {
		return nil, "", 0, false, err
	}

	return msg, filename, di.selected.arrivalIndex, edited, nil
}
func (di *GDDataIndex) selectedMsg_noLock() (*GDLineMsg, error) {
	findex := di.selected.fileIndex
	if findex < 0 || findex >= len(di.files) {
		return nil, fmt.Errorf("bad file index: %v (n=%v)", findex, len(di.files))
	}
	file := di.files[findex]

	lineIndex := di.selected.lineIndex
	if lineIndex < 0 || lineIndex >= len(file.linesMsgs) {
		return nil, fmt.Errorf("bad line index: %v (n=%v)", lineIndex, len(file.linesMsgs))
	}
	lm := file.linesMsgs[lineIndex]

	stepIndex := di.selected.lineStepIndex
	if stepIndex < 0 || stepIndex >= len(lm.lineMsgs) {
		return nil, fmt.Errorf("bad step index: %v (n=%v)", stepIndex, len(lm.lineMsgs))
	}

	return lm.lineMsgs[stepIndex], nil
}

//----------

func (di *GDDataIndex) updateFileEdited_noLock(info *ERowInfo, findex int) bool {
	afd := di.afds[findex]
	edited := !info.EqualToBytesHash(int(afd.FileSize), afd.FileHash)
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
func (di *GDDataIndex) filenameIsIndexed(filename string) bool {
	di.RLock()
	defer di.RUnlock()
	_, ok := di.filesIndexM[filename]
	return ok
}

//----------
//----------
//----------

type GDFileMsgs struct {
	linesMsgs []*GDLineMsgs // [lineIndex] file annotations received

	// current annotation entries to be shown with a file
	annEntries        *drawer4.AnnotationGroup
	annEntriesLMIndex []int // [lineIndex]stepIndex: line messages index: keep selected step index to know the msg entry when coming from a click on an annotation

	//hasNewData bool // performance
}

func NewGDFileMsgs(n int) *GDFileMsgs {
	fms := &GDFileMsgs{
		linesMsgs:         make([]*GDLineMsgs, n),
		annEntries:        drawer4.NewAnnotationGroup(n),
		annEntriesLMIndex: make([]int, n),
	}
	// alloc contiguous memory to slice of pointers
	u := make([]GDLineMsgs, n)
	for i := 0; i < n; i++ {
		fms.linesMsgs[i] = &u[i]
	}
	return fms
}
func (file *GDFileMsgs) findSelectedAndUpdateAnnEntries(arrivalIndex int) (int, int, bool) {
	file.annEntries.Lock()
	defer file.annEntries.Unlock()

	found := false
	selLine := 0
	selLineStep := 0
	for line, lm := range file.linesMsgs {
		k, eqK, foundK := lm.findIndex(arrivalIndex)
		if foundK {
			file.annEntries.Anns[line] = lm.lineMsgs[k].annotation()
			file.annEntriesLMIndex[line] = k
			if eqK {
				found = true
				selLine = line
				selLineStep = k
			}
		} else {
			if len(lm.lineMsgs) > 0 {
				file.annEntries.Anns[line] = lm.lineMsgs[0].emptyAnnotation()
			} else {
				file.annEntries.Anns[line] = nil // no msgs ever received
			}
			file.annEntriesLMIndex[line] = -1
		}
	}
	return selLine, selLineStep, found
}

//----------
//----------
//----------

type GDLineMsgs struct {
	lineMsgs []*GDLineMsg // [arrivalIndex] line annotations received
}

func (lms *GDLineMsgs) findIndex(arrivalIndex int) (int, bool, bool) {
	k := sort.Search(len(lms.lineMsgs), func(i int) bool {
		u := lms.lineMsgs[i].arrivalIndex
		return u >= arrivalIndex
	})
	foundK := false
	eqK := false
	if k < len(lms.lineMsgs) && lms.lineMsgs[k].arrivalIndex == arrivalIndex {
		eqK = true
		foundK = true
	} else {
		k-- // current k is above arrivalIndex, want previous
		foundK = k >= 0
	}
	return k, eqK, foundK
}

//----------
//----------
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
		msg.cache.ann = &drawer4.Annotation{Offset: int(msg.dbgLineMsg.Offset)}
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
