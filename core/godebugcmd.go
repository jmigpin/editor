package core

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/jmigpin/editor/core/godebug"
	"github.com/jmigpin/editor/core/godebug/debug"
	"github.com/jmigpin/editor/core/parseutil"
	"github.com/jmigpin/editor/core/toolbarparser"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/drawutil/drawer3"
	"github.com/pkg/errors"
)

func GoDebugInit(ed *Editor) {
	godebugi = NewGoDebugInstance(ed)
}

func GoDebugCmd(erow *ERow, part *toolbarparser.Part) error {
	args := part.ArgsUnquoted()
	return godebugi.Start(erow, args)
}

func GoDebugStop(ed *Editor) {
	godebugi.CancelAndClear()
}

func GoDebugSelectAnnotation(erow *ERow, annIndex, offset int, typ ui.TASelAnnType) {
	godebugi.SelectAnnotation(erow, annIndex, offset, typ)
}

func GoDebugUpdateUIERowInfo(info *ERowInfo) {
	godebugi.updateUIERowInfo(info)
}

//----------

// Note: Unique instance because there is no easy solution to debug two (or more) programs that have common files.

var godebugi *GoDebugInstance

//----------

type GoDebugInstance struct {
	ed   *Editor
	data struct {
		sync.RWMutex
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

func (gdi *GoDebugInstance) CancelAndClear() {
	gdi.data.Lock()
	gdi.data.dataIndex = nil
	gdi.data.Unlock()
	gdi.cancel()
	gdi.updateUI()
}

//----------

func (gdi *GoDebugInstance) SelectAnnotation(erow *ERow, annIndex, offset int, typ ui.TASelAnnType) {
	if gdi.updateSelectAnnotation(erow, annIndex, offset, typ) {
		gdi.updateUIShowLine(erow)
	}
}

func (gdi *GoDebugInstance) updateSelectAnnotation(erow *ERow, annIndex, offset int, typ ui.TASelAnnType) bool {
	gdi.data.Lock()
	defer gdi.data.Unlock()

	if gdi.data.dataIndex == nil {
		return false
	}

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
		gdi.data.dataIndex.reset()
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
	//s,err := godebug.StringifyItemOffset(msg.DLine.Item, offset) // inner item
	s := godebug.StringifyItemFull(msg.DLine.Item) // full item
	gdi.ed.Messagef("annotation:\n%v\n", s)
}

//----------

func (gdi *GoDebugInstance) currentAnnotationFileLine(erow *ERow, annIndex int) (*GDFileMsgs, *GDLineMsgs, bool) {
	// file
	di := gdi.data.dataIndex
	fi, ok := di.FilesIndex[erow.Info.Name()]
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

func (gdi *GoDebugInstance) Start(erow *ERow, args []string) error {
	// create new erow if necessary
	if erow.Info.IsFileButNotDir() {
		dir := filepath.Dir(erow.Info.Name())
		info := erow.Ed.ReadERowInfo(dir)
		rowPos := erow.Row.PosBelow()
		erow = NewERow(erow.Ed, info, rowPos)
	}

	if !erow.Info.IsDir() {
		return fmt.Errorf("can't run on this erow type")
	}

	// only one instance at a time
	gdi.CancelAndClear() // cancel previous run
	gdi.ready.Lock()     // wait for previous run to finish
	defer gdi.ready.Unlock()

	erow.Exec.Run(func(ctx context.Context, w io.Writer) error {
		// cleanup row content
		erow.Ed.UI.RunOnUIGoRoutine(func() {
			erow.Row.TextArea.SetStrClearHistory("")
		})

		// start data index
		gdi.data.Lock()
		gdi.data.dataIndex = NewGDDataIndex()
		gdi.data.Unlock()

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

	done, err := cmd.Start(ctx, args[1:], nil)
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
	const updatesPerSecond = 20
	var updatec <-chan time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-cmd.Client.Messages:
			//fmt.Fprintf(w, "client msg %#v\n", msg)
			if !ok {
				// last msg (end of program), final ui update
				gdi.updateUI()
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
			updatec = nil
			gdi.updateUI()
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
				return errors.Wrap(err, "request file set positions")
			}
		} else {
			return fmt.Errorf("unhandled string: %v", t)
		}
	case *debug.FilesDataMsg:
		// index data
		if err := gdi.indexMsg(msg); err != nil {
			return err
		}
		// on receiving the filesdatamsg,  send a requeststart
		if err := cmd.RequestStart(); err != nil {
			return errors.Wrap(err, "request start")
		}
	default:
		// index data
		if err := gdi.indexMsg(msg); err != nil {
			return err
		}
	}
	return nil
}

func (gdi *GoDebugInstance) indexMsg(msg interface{}) error {
	gdi.data.Lock()
	defer gdi.data.Unlock()
	return gdi.data.dataIndex.indexMsg(msg)
}

//----------

func (gdi *GoDebugInstance) updateUI() {
	gdi.ed.UI.RunOnUIGoRoutine(func() {
		gdi.data.RLock()
		defer gdi.data.RUnlock()

		gdi.updateUI2()
	})
}

func (gdi *GoDebugInstance) updateUIShowLine(erow *ERow) {
	gdi.ed.UI.RunOnUIGoRoutine(func() {
		gdi.data.RLock()
		defer gdi.data.RUnlock()

		gdi.updateUI2()
		gdi.showSelectedLine(erow)
	})
}

func (gdi *GoDebugInstance) updateUIERowInfo(info *ERowInfo) {
	gdi.ed.UI.RunOnUIGoRoutine(func() {
		gdi.data.RLock()
		defer gdi.data.RUnlock()

		gdi.updateInfoUI(info)
	})
}

//----------

func (gdi *GoDebugInstance) updateUI2() {
	// update all infos (if necessary)
	for _, info := range gdi.ed.ERowInfos {
		gdi.updateInfoUI(info)
	}
}

func (gdi *GoDebugInstance) updateInfoUI(info *ERowInfo) {
	di := gdi.data.dataIndex
	clear := di == nil

	// helper func
	clearDrawerAnnotations := func() {
		for _, erow := range info.ERows {
			ta := erow.Row.TextArea
			if d, ok := ta.Drawer.(*drawer3.PosDrawer); ok {
				if d.Annotations.On() {
					d.Annotations.SetOn(false)
					d.Annotations.Opt.Entries = nil
					ta.MarkNeedsLayoutAndPaint()
				}
			}
		}
	}

	if clear {
		info.UpdateAnnotationsRowState(false)
		info.UpdateAnnotationsEditedRowState(false)
		clearDrawerAnnotations()
	} else {
		findex, ok := di.FilesIndex[info.Name()]
		if !ok {
			info.UpdateAnnotationsRowState(false)
			info.UpdateAnnotationsEditedRowState(false)
			clearDrawerAnnotations()
			return
		}

		info.UpdateAnnotationsRowState(true)

		// check if content has changed
		// TODO: Slow, checking byteshash for each update it !edited
		afd := di.Afds[findex]
		edited := !info.EqualToBytesHash(afd.FileSize, afd.FileHash)
		//edited := info.ERows[0].Row.HasState(ui.RowStateEdited)
		if edited {
			info.UpdateAnnotationsEditedRowState(true)
			clearDrawerAnnotations()
			return
		} else {
			info.UpdateAnnotationsEditedRowState(false)
		}

		di := gdi.data.dataIndex
		file := di.Files[findex]

		// setup lock/unlock each erow annotations
		for _, erow := range info.ERows {
			ta := erow.Row.TextArea
			if d, ok := ta.Drawer.(*drawer3.PosDrawer); ok {
				d.Annotations.Opt.EntriesMu.Lock()
				defer d.Annotations.Opt.EntriesMu.Unlock()
			}
		}

		// update annotations (safe after lock)
		file.updateAnnEntries(di.SelectedArrivalIndex)

		for _, erow := range info.ERows {
			ta := erow.Row.TextArea
			if d, ok := ta.Drawer.(*drawer3.PosDrawer); ok {
				d.Annotations.SetOn(true)
				d.Annotations.Opt.Select.Line = file.SelectedLine
				d.Annotations.Opt.Entries = file.AnnEntries
				ta.MarkNeedsLayoutAndPaint()
			}
		}
	}
}

//----------

// GoDebug data Index
type GDDataIndex struct {
	FilesIndex           map[string]int
	Afds                 []*debug.AnnotatorFileData // file index -> file afd
	Files                []*GDFileMsgs              // file index -> file msgs
	GlobalArrivalIndex   int
	SelectedArrivalIndex int
}

func NewGDDataIndex() *GDDataIndex {
	di := &GDDataIndex{}
	di.FilesIndex = map[string]int{}
	return di
}

func (di *GDDataIndex) reset() {
	di.GlobalArrivalIndex = 0
	di.SelectedArrivalIndex = 0
	for _, f := range di.Files {
		f.reset()
	}
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

func (di *GDDataIndex) indexMsg(msg interface{}) error {
	switch t := msg.(type) {
	case *debug.FilesDataMsg:
		di.Afds = t.Data
		// index filenames
		di.FilesIndex = map[string]int{}
		for _, afd := range di.Afds {
			di.FilesIndex[afd.Filename] = afd.FileIndex
		}
		// init index
		di.Files = make([]*GDFileMsgs, len(di.Afds))
		for _, afd := range di.Afds {
			// check index
			if afd.FileIndex >= len(di.Files) {
				return fmt.Errorf("bad file index at init: %v len=%v", afd.FileIndex, len(di.Files))
			}
			di.Files[afd.FileIndex] = NewGDFileMsgs(afd)
		}
	case *debug.LineMsg:
		// check index
		l1 := len(di.Files)
		if t.FileIndex >= l1 {
			return fmt.Errorf("bad file index: %v len=%v", t.FileIndex, l1)
		}
		// check index
		l2 := len(di.Files[t.FileIndex].Lines)
		if t.DebugIndex >= l2 {
			return fmt.Errorf("bad debug index: %v len=%v", t.DebugIndex, l2)
		}
		// line msg
		lm := &GDLineMsg{GlobalArrivalIndex: di.GlobalArrivalIndex, DLine: t}
		// index msg
		w := &di.Files[t.FileIndex].Lines[t.DebugIndex].Msgs
		*w = append(*w, lm)

		// auto update selected index if at last position
		if di.SelectedArrivalIndex == di.GlobalArrivalIndex-1 {
			di.SelectedArrivalIndex++
		}

		di.GlobalArrivalIndex++

		//// mark as having new data
		//di.Files[t.FileIndex].HasNewData = true

	default:
		return fmt.Errorf("unexpected msg: %T", msg)
	}
	return nil
}

//----------

type GDFileMsgs struct {
	n int // n debug entries (debuglen)

	// all annotations received
	Lines []GDLineMsgs

	// current annotation entries to be shown with a file
	AnnEntries        []*drawer3.Annotation
	AnnEntriesLMIndex []int // line messages index

	SelectedLine int
	//HasNewData bool // performance

}

func NewGDFileMsgs(afd *debug.AnnotatorFileData) *GDFileMsgs {
	file := &GDFileMsgs{n: afd.DebugLen}
	file.reset()
	return file
}

func (file *GDFileMsgs) reset() {
	n := file.n // keep old value
	*file = GDFileMsgs{
		n:                 n,
		SelectedLine:      -1,
		Lines:             make([]GDLineMsgs, n),
		AnnEntries:        make([]*drawer3.Annotation, n),
		AnnEntriesLMIndex: make([]int, n),
	}
}

func (file *GDFileMsgs) updateAnnEntries(maxArrivalIndex int) {
	file.SelectedLine = -1
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
	cachedAnn          *drawer3.Annotation
}

func (msg *GDLineMsg) build() *drawer3.Annotation {
	if msg.cachedAnn == nil {
		msg.cachedAnn = &drawer3.Annotation{Offset: msg.DLine.Offset}
	}
	return msg.cachedAnn
}

func (msg *GDLineMsg) annotation() *drawer3.Annotation {
	ann := msg.build()

	// stringify item
	if msg.itemBytes == nil {
		msg.itemBytes = []byte(godebug.StringifyItem(msg.DLine.Item))
	}
	ann.Bytes = msg.itemBytes

	return ann
}

func (msg *GDLineMsg) emptyAnnotation() *drawer3.Annotation {
	ann := msg.build()
	ann.Bytes = []byte(" ")
	return ann
}
