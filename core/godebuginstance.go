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

//godebug:annotatefile
//godebug:annotatefile:godebug/cmd.go
//godebug:annotatefile:godebug/debug/proto.go

////godebug:annotatefile:editor.go
////godebug:annotatefile:openerow.go
////godebug:annotatefile:../?

// Note: Should have a unique instance because there is no easy solution to debug two (or more) programs that have common files in the same editor

const updatesPerSecond = 12

type GoDebugManager struct {
	ed  *Editor
	gdi struct {
		sync.Mutex
		gdi *GoDebugInstance
	}
}

func NewGoDebugManager(ed *Editor) *GoDebugManager {
	gdm := &GoDebugManager{ed: ed}
	return gdm
}

func (gdm *GoDebugManager) RunAsync(startCtx context.Context, erow *ERow, args []string) error {
	gdm.gdi.Lock()
	defer gdm.gdi.Unlock()

	gdm.cancelAndWaitAndClear2() // previous instance

	// setup instance context
	ctx, cancel := context.WithCancel(context.Background())

	// call cancel if startCtx is done
	clearWatching := ctxutil.WatchDone(startCtx, cancel)
	defer clearWatching()

	gdi, err := newGoDebugInstance(ctx, gdm, erow, args)
	if err != nil {
		return err
	}
	gdm.gdi.gdi = gdi

	return nil
}

//----------

func (gdm *GoDebugManager) CancelAndClear() {
	gdm.gdi.Lock()
	defer gdm.gdi.Unlock()
	gdm.cancelAndWaitAndClear2()
}
func (gdm *GoDebugManager) cancelAndWaitAndClear2() {
	if gdm.gdi.gdi != nil {
		gdm.gdi.gdi.cancelAndWaitAndClear()
		gdm.gdi.gdi = nil
	}
}

//----------

func (gdm *GoDebugManager) SelectAnnotation(rowPos *ui.RowPos, ev *ui.RootSelectAnnotationEvent) {
	gdm.gdi.Lock()
	defer gdm.gdi.Unlock()
	if gdm.gdi.gdi != nil {
		gdm.gdi.gdi.selectAnnotation(rowPos, ev)
	}
}

func (gdm *GoDebugManager) SelectERowAnnotation(erow *ERow, ev *ui.TextAreaSelectAnnotationEvent) {
	gdm.gdi.Lock()
	defer gdm.gdi.Unlock()
	if gdm.gdi.gdi != nil {
		gdm.gdi.gdi.selectERowAnnotation(erow, ev)
	}
}

func (gdm *GoDebugManager) AnnotationFind(s string) error {
	gdm.gdi.Lock()
	defer gdm.gdi.Unlock()
	if gdm.gdi.gdi == nil {
		return fmt.Errorf("missing godebug instance")
	}
	return gdm.gdi.gdi.annotationFind(s)
}

func (gdm *GoDebugManager) UpdateInfoAnnotations(info *ERowInfo) {
	gdm.gdi.Lock()
	defer gdm.gdi.Unlock()
	if gdm.gdi.gdi != nil {
		gdm.gdi.gdi.updateInfoAnnotations(info)
	}
}

func (gdm *GoDebugManager) Trace() error {
	gdm.gdi.Lock()
	defer gdm.gdi.Unlock()
	if gdm.gdi.gdi == nil {
		return fmt.Errorf("missing godebug instance")
	}
	return gdm.gdi.gdi.trace()
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
	ctx    context.Context
	cancel context.CancelFunc
	gdm    *GoDebugManager
	di     *GDDataIndex
}

func newGoDebugInstance(ctx context.Context, gdm *GoDebugManager, erow *ERow, args []string) (*GoDebugInstance, error) {
	gdi := &GoDebugInstance{gdm: gdm}
	gdi.di = NewGDDataIndex(gdi)

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
		return nil, fmt.Errorf("can't run on this erow type")
	}

	_, cancel := erow.Exec.RunAsyncWithCancel(func(erowCtx context.Context, rw io.ReadWriter) error {
		return gdi.runCmd(erowCtx, erow, args, rw)
	})

	// full ctx for the duration of the instance, not just the cmd
	ctx2, cancel2 := context.WithCancel(ctx)
	gdi.ctx = ctx2
	gdi.cancel = func() {
		cancel()  // cancel cmd
		cancel2() // clears resources
	}

	return gdi, nil
}
func (gdi *GoDebugInstance) cancelAndWaitAndClear() {
	gdi.cancel()
	<-gdi.ctx.Done()
	gdi.clearAnnotations()
}

//----------

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
	showLine, err := gdi.selectERowAnnotation2(erow, ev)
	if err != nil {
		//gdi.gdm.printError(err)
		//gdi.updateAnnotations()
		//return
	}
	if showLine {
		gdi.updateAnnotationsAndShowLine(erow, erow.Row.PosBelow())
	}
}
func (gdi *GoDebugInstance) selectERowAnnotation2(erow *ERow, ev *ui.TextAreaSelectAnnotationEvent) (bool, error) {
	switch ev.Type {
	case ui.TasatPrev:
		return true, gdi.di.selectPrev()
	case ui.TasatNext:
		return true, gdi.di.selectNext()
	case ui.TasatMsg,
		ui.TasatMsgPrev,
		ui.TasatMsgNext:
		return true, gdi.di.selectMsgAnnotation(erow.Info.Name(), ev.AnnotationIndex, ev.Type)
	case ui.TasatPrint:
		gdi.printIndex(erow, ev.AnnotationIndex, ev.Offset)
		return false, nil
	case ui.TasatPrintPreviousAll:
		gdi.printIndexAllPrevious(erow, ev.AnnotationIndex, ev.Offset)
		return false, nil
	default:
		return false, fmt.Errorf("todo: %#v", ev)
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

func (gdi *GoDebugInstance) trace() error {
	msgs := gdi.di.trace()

	// build output
	sb := strings.Builder{}
	for i := len(msgs) - 1; i >= 0; i-- { // reverse order
		msg := msgs[i]
		afd := gdi.di.afds[msg.offsetMsg.FileIndex]
		loc := fmt.Sprintf("%v:o=%d", afd.Filename, msg.offsetMsg.Offset)
		s := godebug.StringifyItemFull(msg.offsetMsg.Item)
		u := fmt.Sprintf("%v: %v", s, loc)
		sb.WriteString("\t" + u + "\n")
	}

	gdi.gdm.Printf("trace (%d entries):\n%v\n", len(msgs), sb.String())
	return nil
}

//----------

func (gdi *GoDebugInstance) printIndex(erow *ERow, annIndex, offset int) {
	msg, ok := gdi.di.annMsg(erow.Info.Name(), annIndex)
	if !ok {
		return
	}
	// build output
	s := godebug.StringifyItemFull(msg.offsetMsg.Item)
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
		s := godebug.StringifyItemFull(msg.offsetMsg.Item)
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
		if gdi.ctx.Err() == nil {
			gdi.updateAnnotations()
		}
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
		if err := gdi.handleMsg(v, w, cmd); err != nil {
			handleError(err)
			break
		}
	}
}
func (gdi *GoDebugInstance) handleMsg(msg any, w io.Writer, cmd *godebug.Cmd) error {
	switch t := msg.(type) {
	case *debug.FilesDataMsg:
		//fmt.Fprintf(w, "godebug: index data received\n")
		//gdi.gdm.ed.Messagef(w, "godebug: index data received\n")
		return gdi.di.handleFilesDataMsg(t)
	case *debug.OffsetMsg:
		return gdi.di.handleOffsetMsgs(t)
	case *debug.OffsetMsgs:
		return gdi.di.handleOffsetMsgs(*t...)
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
	entries, selMsgIndex, edited, fileFound := gdi.di.findSelectedAndUpdateAnnEntries(info)

	info.UpdateAnnotationsRowState(fileFound)
	info.UpdateAnnotationsEditedRowState(edited)

	if !fileFound || edited {
		gdi.clearInfoAnnotations2(info)
		return
	}

	// set annotations into opened (existing) erows
	// Note: the current selected debug line might not have an open erow (ex: when auto increased to match the lastarrivalindex).
	for _, erow := range info.ERows {
		gdi.setAnnotations(erow, true, selMsgIndex, entries)
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
			Offset:   int(msg.offsetMsg.Offset),
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

	resetCount       int // number of resets to number msgs
	lastArrivalIndex int
	selected         struct {
		arrivalIndex int

		// TODO: review, after reset(), what is the state of this
		fileIndex    int
		msgIndex     int
		msgStepIndex int
	}
}

func NewGDDataIndex(gdi *GoDebugInstance) *GDDataIndex {
	di := &GDDataIndex{gdi: gdi}
	di.filesIndexM = map[string]int{}
	di.filesEdited = map[int]bool{}
	di.resetArrivalIndex()
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

//----------

func (di *GDDataIndex) reset() {
	di.Lock()
	defer di.Unlock()
	di.reset2()
}
func (di *GDDataIndex) reset2() {
	di.resetCount++
	di.resetArrivalIndex()
	for _, f := range di.files {
		n := len(f.msgs) // keep n
		u := NewGDFileMsgs(n)
		*f = *u
	}
}
func (di *GDDataIndex) resetArrivalIndex() {
	di.lastArrivalIndex = -1
	di.selected.arrivalIndex = di.lastArrivalIndex
}

//----------

func (di *GDDataIndex) handleFilesDataMsg(fdm *debug.FilesDataMsg) error {
	di.Lock()
	defer di.Unlock()

	di.reset2()

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
		di.files[afd.FileIndex] = NewGDFileMsgs(int(afd.NMsgIndexes))
	}
	return nil
}

func (di *GDDataIndex) handleOffsetMsgs(msgs ...*debug.OffsetMsg) error {
	di.Lock()
	defer di.Unlock()
	for _, msg := range msgs {
		err := di.handleOffsetMsg_noLock(msg)
		if err != nil {
			return err
		}
	}
	return nil
}
func (di *GDDataIndex) handleOffsetMsg_noLock(u *debug.OffsetMsg) error {
	// check index
	l1 := len(di.files)
	if int(u.FileIndex) >= l1 {
		return fmt.Errorf("bad file index: %v len=%v", u.FileIndex, l1)
	}
	// check index
	l2 := len(di.files[int(u.FileIndex)].msgs)
	if int(u.MsgIndex) >= l2 {
		return fmt.Errorf("bad debug index: %v len=%v", u.MsgIndex, l2)
	}
	// msg
	di.lastArrivalIndex++ // starts/clears to -1, so first n is 0
	lm := &GDOffsetMsg{arrivalIndex: di.lastArrivalIndex, resetIndex: di.resetCount, offsetMsg: u}
	// append newly arrived msg
	w := &di.files[u.FileIndex].msgs[u.MsgIndex].arrivals
	*w = append(*w, lm)

	// auto update selected index if at last position
	if di.selected.arrivalIndex == di.lastArrivalIndex-1 {
		di.selected.arrivalIndex = di.lastArrivalIndex
	}

	return nil
}

//----------

func (di *GDDataIndex) annMsg(filename string, annIndex int) (*GDOffsetMsg, bool) {
	di.RLock()
	defer di.RUnlock()

	file, line, ok := di.msgIndexFileMsg_noLock(filename, annIndex)
	if !ok {
		return nil, false
	}
	// current msg index at line
	k := file.annsMsgIndex[annIndex]      // same length as lineMsgs
	if k < 0 || k >= len(line.arrivals) { // currently nothing is shown or cleared
		return nil, false
	}
	return line.arrivals[k], true
}

func (di *GDDataIndex) annPreviousMsgs(filename string, annIndex int) ([]*GDOffsetMsg, bool) {
	di.RLock()
	defer di.RUnlock()

	file, line, ok := di.msgIndexFileMsg_noLock(filename, annIndex)
	if !ok {
		return nil, false
	}
	// current msg index at line
	k := file.annsMsgIndex[annIndex]      // same length as lineMsgs
	if k < 0 || k >= len(line.arrivals) { // currently nothing is shown or cleared
		return nil, false
	}
	return line.arrivals[:k+1], true
}

func (di *GDDataIndex) selectedAnnFind(s string) (*GDOffsetMsg, bool) {
	di.RLock()
	defer di.RUnlock()

	annIndex, filename, ok := di.selectedArrivalIndexFilename()
	if !ok {
		return nil, false
	}

	file, msg, ok := di.msgIndexFileMsg_noLock(filename, annIndex)
	if !ok {
		return nil, false
	}

	b := []byte(s)
	k := file.annsMsgIndex[annIndex] // current entry
	for i := 0; i < len(msg.arrivals); i++ {
		h := (k + 1 + i) % len(msg.arrivals)
		om := msg.arrivals[h]
		ann := om.annotation()
		j := bytes.Index(ann.Bytes, b)
		if j >= 0 {
			di.selected.arrivalIndex = om.arrivalIndex
			return om, true
		}
	}

	return nil, false
}
func (di *GDDataIndex) selectMsgAnnotation(filename string, msgIndex int, typ ui.TASelAnnType) error {
	di.Lock() // writes di.selected
	defer di.Unlock()

	file, msg, ok := di.msgIndexFileMsg_noLock(filename, msgIndex)
	if !ok {
		return fmt.Errorf("file not indexed: %v", filename)
	}

	if len(msg.arrivals) == 0 {
		return fmt.Errorf("no msgs in this line yet")
	}

	// current msg index at line
	k := file.annsMsgIndex[msgIndex] // same length as lineMsgs

	// annotation already selected before attempting to change
	selected := k >= 0 && k < len(msg.arrivals) && msg.arrivals[k].arrivalIndex == di.selected.arrivalIndex

	// no line selected yet on this line
	if k < 0 {
		k = 0 // NOTE: there is at least one line (tested above)
	}

	// from here: k>=0

	// adjust k according to type
	switch typ {
	case ui.TasatMsg:
		// might be selected already or not
	case ui.TasatMsgPrev:
		if k == 0 {
			if selected {
				return fmt.Errorf("already at line first index")
			}
		} else {
			k--
		}
	case ui.TasatMsgNext:
		if k >= len(msg.arrivals)-1 {
			if selected {
				return fmt.Errorf("already at line last index")
			}
		} else {
			k++
		}
	default:
		panic(fmt.Sprintf("unexpected type: %v", typ))
	}

	di.selected.arrivalIndex = msg.arrivals[k].arrivalIndex
	return nil
}
func (di *GDDataIndex) msgIndexFileMsg_noLock(filename string, msgIndex int) (*GDFileMsgs, *GDMsg, bool) {
	// file
	findex, ok := di.FilesIndex(filename)
	if !ok {
		return nil, nil, false
	}
	file := di.files[findex]
	// msg
	if msgIndex < 0 || msgIndex >= len(file.msgs) {
		return nil, nil, false
	}

	return file, file.msgs[msgIndex], true
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

func (di *GDDataIndex) findSelectedAndUpdateAnnEntries(info *ERowInfo) (entries *drawer4.AnnotationGroup, selMsgIndex int, edited bool, fileFound bool) {
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

	if selMsgIndex, ok = di.findSelectedAndUpdateAnnEntries_noLock(findex); !ok {
		selMsgIndex = -1
	}
	return di.files[findex].anns, selMsgIndex, edited, true
}

func (di *GDDataIndex) findSelectedAndUpdateAnnEntries_noLock(findex int) (int, bool) {
	file := di.files[findex]
	selMsg, selMsgStep, selFound := file.findSelectedAndUpdateAnnEntries(di.selected.arrivalIndex)
	if selFound {
		di.selected.fileIndex = findex
		di.selected.msgIndex = selMsg
		di.selected.msgStepIndex = selMsgStep
	}
	return selMsg, selFound
}

//----------

func (di *GDDataIndex) selectedMsg() (*GDOffsetMsg, string, int, bool, error) {
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
func (di *GDDataIndex) selectedMsg_noLock() (*GDOffsetMsg, error) {
	findex := di.selected.fileIndex
	if findex < 0 || findex >= len(di.files) {
		return nil, fmt.Errorf("bad file index: %v (n=%v)", findex, len(di.files))
	}
	file := di.files[findex]

	msgIndex := di.selected.msgIndex
	if msgIndex < 0 || msgIndex >= len(file.msgs) {
		return nil, fmt.Errorf("bad line index: %v (n=%v)", msgIndex, len(file.msgs))
	}
	m := file.msgs[msgIndex]

	stepIndex := di.selected.msgStepIndex
	if stepIndex < 0 || stepIndex >= len(m.arrivals) {
		return nil, fmt.Errorf("bad step index: %v (n=%v)", stepIndex, len(m.arrivals))
	}

	return m.arrivals[stepIndex], nil
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
		for j, lm := range file.msgs {
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

func (di *GDDataIndex) trace() []*GDOffsetMsg {
	di.RLock()
	defer di.RUnlock()

	res := []*GDOffsetMsg{}

	arrivalIndex := di.selected.arrivalIndex

	// all files, all lines, if currently holding, keep it; check arrival
	for _, f := range di.files {
		for _, m := range f.msgs {
			idx, eq, found := m.findIndex(arrivalIndex)
			if !found {
				continue
			}
			om := m.arrivals[idx]
			// current line
			if eq {
				res = append(res, om)
				continue
			}
			// call lines
			switch om.offsetMsg.Item.(type) {
			case *debug.ItemCallEnter,
				*debug.ItemUnaryEnter,
				*debug.ItemSend:
				res = append(res, om)
			}
		}
	}

	return res
}

//----------
//----------
//----------

type GDFileMsgs struct {
	msgs []*GDMsg // [msgIndex] annotations received

	// current annotation entries to be shown with a file
	anns         *drawer4.AnnotationGroup
	annsMsgIndex []int // [msgIndex]stepIndex: msgs index: keep selected step index to know the msg entry when coming from a click on an annotation
}

func NewGDFileMsgs(n int) *GDFileMsgs {
	fms := &GDFileMsgs{
		msgs:         make([]*GDMsg, n),
		anns:         drawer4.NewAnnotationGroup(n),
		annsMsgIndex: make([]int, n),
	}
	// alloc contiguous memory to slice of pointers
	u := make([]GDMsg, n)
	for i := 0; i < n; i++ {
		fms.msgs[i] = &u[i]
	}
	return fms
}
func (file *GDFileMsgs) findSelectedAndUpdateAnnEntries(arrivalIndex int) (int, int, bool) {
	file.anns.Lock()
	defer file.anns.Unlock()

	found := false
	selMsg := 0
	selMsgStep := 0
	for h, m := range file.msgs {
		k, eqK, foundK := m.findIndex(arrivalIndex)
		if foundK {
			file.anns.Anns[h] = m.arrivals[k].annotation()
			file.annsMsgIndex[h] = k
			if eqK {
				found = true
				selMsg = h
				selMsgStep = k
			}
		} else {
			if len(m.arrivals) > 0 {
				file.anns.Anns[h] = m.arrivals[0].emptyAnnotation()
			} else {
				file.anns.Anns[h] = nil // no msgs ever received
			}
			file.annsMsgIndex[h] = -1
		}
	}
	return selMsg, selMsgStep, found
}

//----------
//----------
//----------

type GDMsg struct {
	arrivals []*GDOffsetMsg // [arrivalIndex] annotations received
}

func (u *GDMsg) findIndex(arrivalIndex int) (int, bool, bool) {
	k := sort.Search(len(u.arrivals), func(i int) bool {
		u := u.arrivals[i].arrivalIndex
		return u >= arrivalIndex
	})
	foundK := false
	eqK := false
	if k < len(u.arrivals) && u.arrivals[k].arrivalIndex == arrivalIndex {
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

type GDOffsetMsg struct {
	arrivalIndex int
	resetIndex   int
	offsetMsg    *debug.OffsetMsg
	cache        struct {
		ann   *drawer4.Annotation
		empty *drawer4.Annotation
	}
}

func (msg *GDOffsetMsg) annotation() *drawer4.Annotation {
	if msg.cache.ann != nil {
		return msg.cache.ann
	}

	ann := &drawer4.Annotation{}
	ann.Offset = int(msg.offsetMsg.Offset)

	s := godebug.StringifyItem(msg.offsetMsg.Item)
	ann.Bytes = []byte(s)

	s2 := ""
	if msg.resetIndex >= 2 {
		s2 = fmt.Sprintf("%d:", msg.resetIndex)
	}
	s3 := fmt.Sprintf("#%s%d", s2, msg.arrivalIndex)
	ann.NotesBytes = []byte(s3)

	msg.cache.ann = ann

	return ann
}
func (msg *GDOffsetMsg) emptyAnnotation() *drawer4.Annotation {
	if msg.cache.empty != nil {
		return msg.cache.empty
	}

	ann := &drawer4.Annotation{}
	ann.Offset = int(msg.offsetMsg.Offset)
	ann.Bytes = []byte(" ") // allow a clickable rune (empty space)
	ann.NotesBytes = nil

	msg.cache.empty = ann

	return ann
}
