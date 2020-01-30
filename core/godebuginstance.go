package core

import (
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
	"github.com/jmigpin/editor/core/parseutil"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/drawutil/drawer4"
)

const updatesPerSecond = 15

//----------

// Note: Should have a unique instance because there is no easy solution to debug two (or more) programs that have common files in the same editor

type GoDebugInstance struct {
	ed   *Editor
	data struct {
		mu        sync.RWMutex
		dataIndex *GDDataIndex
	}
	cancel context.CancelFunc
	ready  sync.Mutex
}

func NewGoDebugInstance(ed *Editor) *GoDebugInstance {
	gdi := &GoDebugInstance{ed: ed}
	gdi.cancel = func() {}
	return gdi
}

//----------

func (gdi *GoDebugInstance) dataLock() bool {
	gdi.data.mu.Lock()
	if gdi.data.dataIndex == nil {
		gdi.data.mu.Unlock()
		return false
	}
	return true
}

func (gdi *GoDebugInstance) dataUnlock() {
	gdi.data.mu.Unlock()
}

func (gdi *GoDebugInstance) dataRLock() bool {
	gdi.data.mu.RLock()
	if gdi.data.dataIndex == nil {
		gdi.data.mu.RUnlock()
		return false
	}
	return true
}

func (gdi *GoDebugInstance) dataRUnlock() {
	gdi.data.mu.RUnlock()
}

//----------

func (gdi *GoDebugInstance) CancelAndClear() {
	if !gdi.dataLock() {
		return
	}
	gdi.data.dataIndex = nil
	gdi.clearInfosUI()
	gdi.dataUnlock()

	gdi.cancel()
}

//----------

func (gdi *GoDebugInstance) SelectAnnotation(erow *ERow, annIndex, offset int, typ ui.TASelAnnType) {
	if gdi.updateSelectAnnotation(erow, annIndex, offset, typ) {
		gdi.updateUIShowLine(erow)
	}
}

func (gdi *GoDebugInstance) updateSelectAnnotation(erow *ERow, annIndex, offset int, typ ui.TASelAnnType) bool {
	if !gdi.dataLock() {
		return false
	}
	defer gdi.dataUnlock()

	update := false
	switch typ {
	case ui.TASelAnnTypeCurrent,
		ui.TASelAnnTypeCurrentPrev,
		ui.TASelAnnTypeCurrentNext:
		update = gdi.selectCurrent(erow, annIndex, offset, typ)
	case ui.TASelAnnTypePrev:
		update = gdi.selectPrev()
	case ui.TASelAnnTypeNext:
		update = gdi.selectNext()
	case ui.TASelAnnTypeLast:
		update = gdi.selectLast()
	case ui.TASelAnnTypeClear:
		update = true
		gdi.data.dataIndex.clearMsgs()
	case ui.TASelAnnTypePrint:
		gdi.printIndex(erow, annIndex, offset)
	}

	return update
}

func (gdi *GoDebugInstance) selectCurrent(erow *ERow, annIndex, offset int, typ ui.TASelAnnType) bool {
	file, line, ok := gdi.currentAnnotationFileLine(erow, annIndex)
	if !ok {
		return false
	}

	k := file.AnnEntriesLMIndex[annIndex]

	// currently nothing is shown, use first
	if k < 0 {
		k = 0
	}

	// adjust k according to type
	switch typ {
	case ui.TASelAnnTypeCurrent: // use k, nothing todo
	case ui.TASelAnnTypeCurrentPrev:
		if k == 0 {
			return false
		}
		k--
	case ui.TASelAnnTypeCurrentNext:
		if k >= len(line.Msgs)-1 {
			return false
		}
		k++
	default:
		panic(fmt.Sprintf("unexpected type: %v", typ))
	}

	// set selected index
	di := gdi.data.dataIndex
	di.SelectedArrivalIndex = line.Msgs[k].GlobalArrivalIndex

	return true
}

func (gdi *GoDebugInstance) selectNext() bool {
	di := gdi.data.dataIndex
	if di.SelectedArrivalIndex < di.GlobalArrivalIndex-1 {
		di.SelectedArrivalIndex++
		gdi.openArrivalIndexERow()
		return true
	}
	return false
}

func (gdi *GoDebugInstance) selectPrev() bool {
	di := gdi.data.dataIndex
	if di.SelectedArrivalIndex > 0 {
		di.SelectedArrivalIndex--
		gdi.openArrivalIndexERow()
		return true
	}
	return false
}

func (gdi *GoDebugInstance) selectLast() bool {
	di := gdi.data.dataIndex
	if di.SelectedArrivalIndex < di.GlobalArrivalIndex-1 {
		di.SelectedArrivalIndex = di.GlobalArrivalIndex - 1
	}
	// show always
	gdi.openArrivalIndexERow()
	return true
}

//----------

func (gdi *GoDebugInstance) printIndex(erow *ERow, annIndex, offset int) {
	file, line, ok := gdi.currentAnnotationFileLine(erow, annIndex)
	if !ok {
		return
	}

	// current msg index at line
	k := file.AnnEntriesLMIndex[annIndex]
	if k < 0 { // currently nothing is shown
		return
	}

	// msg
	msg := line.Msgs[k]

	// output
	//s := godebug.StringifyItemOffset(msg.DLine.Item, offset) // inner item
	s := godebug.StringifyItemFull(msg.DLine.Item) // full item
	gdi.ed.Messagef("annotation:\n\t%v\n", s)
}

//----------

func (gdi *GoDebugInstance) currentAnnotationFileLine(erow *ERow, annIndex int) (*GDFileMsgs, *GDLineMsgs, bool) {
	// file
	di := gdi.data.dataIndex
	fi, ok := di.FilesIndex(erow.Info.Name())
	if !ok {
		return nil, nil, false
	}
	file := di.Files[fi]
	if annIndex < 0 || annIndex >= len(file.AnnEntriesLMIndex) {
		return nil, nil, false
	}
	// line
	return file, &file.Lines[annIndex], true
}

//----------

func (gdi *GoDebugInstance) openArrivalIndexERow() {
	di := gdi.data.dataIndex
	filename, ok := di.selectedArrivalIndexFilename(di.SelectedArrivalIndex)
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
	OpenFileERow(gdi.ed, conf)
}

//----------

func (gdi *GoDebugInstance) Start(erow *ERow, args []string) error {
	// warn other annotators about starting a godebug session
	ta := erow.Row.TextArea
	_ = gdi.ed.CanModifyAnnotations(EdAnnReqGoDebug, ta, "starting_session")

	// create new erow if necessary
	if erow.Info.IsFileButNotDir() {
		dir := filepath.Dir(erow.Info.Name())
		info := erow.Ed.ReadERowInfo(dir)
		rowPos := erow.Row.PosBelow()
		erow = NewERow(erow.Ed, info, rowPos)
	}

	if !erow.Info.IsDir() {
		// TODO: erow type should have a run cmd func that transfers this
		return fmt.Errorf("can't run on this erow type")
	}

	// only one instance at a time
	gdi.CancelAndClear() // cancel previous run

	erow.Exec.Start(func(ctx context.Context, w io.Writer) error {
		// wait for previous run to finish
		gdi.ready.Lock()
		defer gdi.ready.Unlock()

		// cleanup row content
		erow.Ed.UI.RunOnUIGoRoutine(func() {
			erow.Row.TextArea.SetStrClearHistory("")
		})

		// start data index
		gdi.data.mu.Lock()
		gdi.data.dataIndex = NewGDDataIndex(gdi.ed)
		gdi.data.mu.Unlock()

		// keep ctx cancel to be able to stop if necessary
		ctx2, cancel := context.WithCancel(ctx)
		defer cancel() // can't defer gdi.cancel here (concurrency)
		gdi.cancel = cancel

		gdi.updateUI()

		return gdi.start2(erow, args, ctx2, w)
	})

	return nil
}

func (gdi *GoDebugInstance) start2(erow *ERow, args []string, ctx context.Context, w io.Writer) error {
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

	// handle client msgs loop (blocking)
	gdi.clientMsgsLoop(ctx, w, cmd)

	return cmd.Wait()
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
		if err := gdi.handleFilesDataMsg(t); err != nil {
			return err
		}
		// on receiving the filesdatamsg, send a requeststart
		if err := cmd.RequestStart(); err != nil {
			return fmt.Errorf("request start: %w", err)
		}
	case *debug.LineMsg:
		return gdi.handleLineMsg(t)
	case []*debug.LineMsg:
		return gdi.handleLineMsgs(t)
	default:
		return fmt.Errorf("unexpected msg: %T", msg)
	}
	return nil
}

func (gdi *GoDebugInstance) handleFilesDataMsg(msg *debug.FilesDataMsg) error {
	if !gdi.dataLock() {
		return fmt.Errorf("dataindex is nil")
	}
	defer gdi.dataUnlock()

	return gdi.data.dataIndex.handleFilesDataMsg(msg)
}

func (gdi *GoDebugInstance) handleLineMsg(msg *debug.LineMsg) error {
	if !gdi.dataLock() {
		return fmt.Errorf("dataindex is nil")
	}
	defer gdi.dataUnlock()

	return gdi.data.dataIndex.handleLineMsg(msg)
}

func (gdi *GoDebugInstance) handleLineMsgs(msgs []*debug.LineMsg) error {
	if !gdi.dataLock() {
		return fmt.Errorf("dataindex is nil")
	}
	defer gdi.dataUnlock()

	for _, msg := range msgs {
		err := gdi.data.dataIndex.handleLineMsg(msg)
		if err != nil {
			return err
		}
	}
	return nil
}

//----------

func (gdi *GoDebugInstance) updateUI() {
	gdi.ed.UI.RunOnUIGoRoutine(func() {
		if !gdi.dataRLock() {
			return
		}
		defer gdi.dataRUnlock()

		gdi.updateUI2()
	})
}

func (gdi *GoDebugInstance) updateUIShowLine(erow *ERow) {
	gdi.ed.UI.RunOnUIGoRoutine(func() {
		if !gdi.dataRLock() {
			return
		}
		defer gdi.dataRUnlock()

		gdi.updateUI2()
		gdi.showSelectedLine(erow)
	})
}

func (gdi *GoDebugInstance) UpdateUIERowInfo(info *ERowInfo) {
	gdi.ed.UI.RunOnUIGoRoutine(func() {
		if !gdi.dataRLock() {
			return
		}
		defer gdi.dataRUnlock()

		gdi.updateInfoUI(info)
	})
}

//----------

func (gdi *GoDebugInstance) clearInfosUI() {
	for _, info := range gdi.ed.ERowInfos() {
		gdi.clearInfoUI(info)
	}
}

func (gdi *GoDebugInstance) clearInfoUI(info *ERowInfo) {
	info.UpdateAnnotationsRowState(false)
	info.UpdateAnnotationsEditedRowState(false)
	gdi.clearDrawerAnn(info)
}

//----------

func (gdi *GoDebugInstance) updateUI2() {
	// update all infos
	for _, info := range gdi.ed.ERowInfos() {
		gdi.updateInfoUI(info)
	}
}

func (gdi *GoDebugInstance) updateInfoUI(info *ERowInfo) {
	di := gdi.data.dataIndex

	findex, ok := di.FilesIndex(info.Name())
	if !ok {
		info.UpdateAnnotationsRowState(false)
		info.UpdateAnnotationsEditedRowState(false)
		gdi.clearDrawerAnn(info)
		return
	}

	info.UpdateAnnotationsRowState(true)

	file := di.Files[findex]
	file.prepareForUpdate() // resets selected line

	// check if content has changed
	afd := di.Afds[findex]
	edited := !info.EqualToBytesHash(afd.FileSize, afd.FileHash)
	if edited {
		info.UpdateAnnotationsEditedRowState(true)
		gdi.clearDrawerAnn(info)
		return
	}

	info.UpdateAnnotationsEditedRowState(false)

	// update annotations (safe after lock)
	file.updateAnnEntries(di.SelectedArrivalIndex)

	for _, erow := range info.ERows {
		gdi.setAnnotations(erow, true, file.SelectedLine, file.AnnEntries)
	}
}

func (gdi *GoDebugInstance) clearDrawerAnn(info *ERowInfo) {
	for _, erow := range info.ERows {
		gdi.setAnnotations(erow, false, 0, nil)
	}
}

func (gdi *GoDebugInstance) setAnnotations(erow *ERow, on bool, selIndex int, entries []*drawer4.Annotation) {
	ta := erow.Row.TextArea
	gdi.ed.SetAnnotations(EdAnnReqGoDebug, ta, on, selIndex, entries)
}

//----------

func (gdi *GoDebugInstance) showSelectedLine(erow *ERow) {
	di := gdi.data.dataIndex
	for _, afd := range di.Afds {
		file := di.Files[afd.FileIndex]

		if file.SelectedLine >= 0 {
			lm := file.Lines[file.SelectedLine]
			if len(lm.Msgs) == 0 {
				continue
			}

			// file offset
			dlm := lm.Msgs[0].DLine
			fo := &parseutil.FilePos{Filename: afd.Filename, Offset: dlm.Offset}

			// show line
			rowPos := erow.Row.PosBelow()
			conf := &OpenFileERowConfig{
				FilePos:             fo,
				RowPos:              rowPos,
				FlashVisibleOffsets: true,
				NewIfNotExistent:    true,
			}
			OpenFileERow(gdi.ed, conf)
		}
	}
}

//----------

// GoDebug data Index
type GDDataIndex struct {
	ed          *Editor
	filesIndexM map[string]int

	GlobalArrivalIndex   int
	SelectedArrivalIndex int

	Afds  []*debug.AnnotatorFileData // file index -> file afd
	Files []*GDFileMsgs              // file index -> file msgs
}

func NewGDDataIndex(ed *Editor) *GDDataIndex {
	di := &GDDataIndex{ed: ed}
	di.filesIndexM = map[string]int{}
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
	for _, f := range di.Files {
		n := len(f.Lines) // keep n
		u := NewGDFileMsgs(n)
		*f = *u
	}
	di.GlobalArrivalIndex = 0
	di.SelectedArrivalIndex = 0
}

//----------

func (di *GDDataIndex) selectedArrivalIndexFilename(arrivalIndex int) (string, bool) {
	for i, f := range di.Files {
		for _, lm := range f.Lines {
			k := sort.Search(len(lm.Msgs), func(i int) bool {
				u := lm.Msgs[i].GlobalArrivalIndex
				return u > arrivalIndex
			})
			k--
			if k >= 0 {
				if lm.Msgs[k].GlobalArrivalIndex == arrivalIndex {
					return di.Afds[i].Filename, true
				}
			}
		}
	}
	return "", false
}

//----------

func (di *GDDataIndex) handleFilesDataMsg(fdm *debug.FilesDataMsg) error {
	di.Afds = fdm.Data
	// index filenames
	di.filesIndexM = map[string]int{}
	for _, afd := range di.Afds {
		name := di.FilesIndexKey(afd.Filename)
		di.filesIndexM[name] = afd.FileIndex
	}
	// init index
	di.Files = make([]*GDFileMsgs, len(di.Afds))
	for _, afd := range di.Afds {
		// check index
		if afd.FileIndex >= len(di.Files) {
			return fmt.Errorf("bad file index at init: %v len=%v", afd.FileIndex, len(di.Files))
		}
		di.Files[afd.FileIndex] = NewGDFileMsgs(afd.DebugLen)
	}
	return nil
}

func (di *GDDataIndex) handleLineMsg(u *debug.LineMsg) error {
	// check index
	l1 := len(di.Files)
	if u.FileIndex >= l1 {
		return fmt.Errorf("bad file index: %v len=%v", u.FileIndex, l1)
	}
	// check index
	l2 := len(di.Files[u.FileIndex].Lines)
	if u.DebugIndex >= l2 {
		return fmt.Errorf("bad debug index: %v len=%v", u.DebugIndex, l2)
	}
	// line msg
	lm := &GDLineMsg{GlobalArrivalIndex: di.GlobalArrivalIndex, DLine: u}
	// index msg
	w := &di.Files[u.FileIndex].Lines[u.DebugIndex].Msgs
	*w = append(*w, lm)

	// auto update selected index if at last position
	if di.SelectedArrivalIndex == di.GlobalArrivalIndex-1 {
		di.SelectedArrivalIndex++
	}

	di.GlobalArrivalIndex++

	//// mark as having new data
	//di.Files[t.FileIndex].HasNewData = true

	return nil
}

//----------

type GDFileMsgs struct {
	// all annotations received
	Lines []GDLineMsgs

	// current annotation entries to be shown with a file
	AnnEntries        []*drawer4.Annotation
	AnnEntriesLMIndex []int // line messages index

	SelectedLine int

	//HasNewData bool // performance
}

func NewGDFileMsgs(n int) *GDFileMsgs {
	return &GDFileMsgs{
		SelectedLine:      -1,
		Lines:             make([]GDLineMsgs, n),
		AnnEntries:        make([]*drawer4.Annotation, n),
		AnnEntriesLMIndex: make([]int, n),
	}
}

func (file *GDFileMsgs) prepareForUpdate() {
	file.SelectedLine = -1
}

func (file *GDFileMsgs) updateAnnEntries(maxArrivalIndex int) {
	for line, lm := range file.Lines {
		k := sort.Search(len(lm.Msgs), func(i int) bool {
			u := lm.Msgs[i].GlobalArrivalIndex
			return u > maxArrivalIndex
		})
		// get less or equal then maxarrivalindex
		k--
		if k < 0 {
			file.AnnEntries[line] = nil
			if len(lm.Msgs) > 0 {
				file.AnnEntries[line] = lm.Msgs[0].emptyAnnotation()
			}
		} else {
			file.AnnEntries[line] = lm.Msgs[k].annotation()

			// selected line
			if lm.Msgs[k].GlobalArrivalIndex == maxArrivalIndex {
				file.SelectedLine = line
			}
		}

		// keep selected k to know the msg entry when coming from a click on an annotation
		file.AnnEntriesLMIndex[line] = k
	}
}

//----------

type GDLineMsgs struct {
	Msgs []*GDLineMsg
}

//----------

type GDLineMsg struct {
	GlobalArrivalIndex int
	DLine              *debug.LineMsg
	itemBytes          []byte
	cachedAnn          *drawer4.Annotation
}

func (msg *GDLineMsg) build() *drawer4.Annotation {
	if msg.cachedAnn == nil {
		msg.cachedAnn = &drawer4.Annotation{Offset: msg.DLine.Offset}
	}
	return msg.cachedAnn
}

func (msg *GDLineMsg) annotation() *drawer4.Annotation {
	ann := msg.build()

	// stringify item
	if msg.itemBytes == nil {
		msg.itemBytes = []byte(godebug.StringifyItem(msg.DLine.Item))
	}
	ann.Bytes = msg.itemBytes

	return ann
}

func (msg *GDLineMsg) emptyAnnotation() *drawer4.Annotation {
	ann := msg.build()
	ann.Bytes = []byte(" ")
	return ann
}
