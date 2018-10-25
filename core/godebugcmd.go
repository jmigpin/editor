package core

import (
	"context"
	"fmt"
	"io"
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
	ready  chan struct{}
}

func NewGoDebugInstance(ed *Editor) *GoDebugInstance {
	gdi := &GoDebugInstance{ed: ed}
	gdi.cancel = func() {}
	gdi.ready = make(chan struct{}, 1)
	gdi.ready <- struct{}{}
	return gdi
}

func (gdi *GoDebugInstance) Start(erow *ERow, args []string) error {
	switch args[1] {
	case "run", "test":
		return gdi.run(erow, args)
	default:
		return fmt.Errorf("expecting {run,test}")
		//return fmt.Errorf("expecting {run,test,find,stop}")
	}
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
	}

	return update
}

func (gdi *GoDebugInstance) selectCurrent(erow *ERow, annIndex, offset int, typ ui.TASelAnnType) bool {
	di := gdi.data.dataIndex

	fi, ok := di.FilesIndex[erow.Info.Name()]
	if !ok {
		return false
	}

	fmsgs := &di.FileMsgs[fi]

	if annIndex < 0 || annIndex >= len(fmsgs.AnnEntriesLMIndex) {
		return false
	}

	lm := fmsgs.LineMsgs[annIndex]
	k := fmsgs.AnnEntriesLMIndex[annIndex]

	// currently nothing is shown, use first
	if k < 0 {
		k = 0
	}

	// adjust k according to type
	switch typ {
	case ui.TASelAnnTypeCurrent: // use k, nothing todo
	case ui.TASelAnnTypeCurrentPrev:
		if k > 0 {
			k--
		}
	case ui.TASelAnnTypeCurrentNext:
		if k < len(lm.Msgs)-1 {
			k++
		}
	}

	// set selected index
	di.SelectedArrivalIndex = lm.Msgs[k].GlobalArrivalIndex

	return true
}

func (gdi *GoDebugInstance) selectNext() bool {
	// TODO: find next with open erow

	di := gdi.data.dataIndex
	if di.SelectedArrivalIndex < di.GlobalArrivalIndex-1 {
		di.SelectedArrivalIndex++
		return true
	}
	return false
}

func (gdi *GoDebugInstance) selectPrev() bool {
	// TODO: find next with open erow

	di := gdi.data.dataIndex
	if di.SelectedArrivalIndex > 0 {
		di.SelectedArrivalIndex--
		return true
	}
	return false
}

//----------

func (gdi *GoDebugInstance) showSelectedLine(erow *ERow) {
	di := gdi.data.dataIndex
	for i := 0; i < len(di.FileMsgs); i++ {
		fmsgs := &di.FileMsgs[i]
		afd := di.Afds[i]

		if fmsgs.SelectedLine >= 0 {
			lm := fmsgs.LineMsgs[fmsgs.SelectedLine]
			if len(lm.Msgs) == 0 {
				continue
			}

			dlm := lm.Msgs[0].DLineMsg

			// show line
			rowPos := erow.Row.PosBelow()
			fo := &parseutil.FileOffset{Filename: afd.Filename, Offset: dlm.Offset}
			conf := &OpenFileERowConfig{
				FileOffset:          fo,
				RowPos:              rowPos,
				FlashVisibleOffsets: true,
				NewIfNotExistent:    true,
			}
			OpenFileERow(gdi.ed, conf)
		}
	}
}

//----------

func (gdi *GoDebugInstance) CheckTextAreaContent(erow *ERow) {
	// update godebug annotations if hash doesn't match
	// TODO
}

//----------

func (gdi *GoDebugInstance) run(erow *ERow, args []string) error {
	if !erow.Info.IsDir() {
		return fmt.Errorf("not a directory")
	}

	erow.Row.TextArea.SetStrClearHistory("")

	// start/end ready
	<-gdi.ready
	defer func() { gdi.ready <- struct{}{} }()

	erow.Exec.Run(func(ctx context.Context, w io.Writer) error {

		// start data index
		gdi.data.Lock()
		gdi.data.dataIndex = NewGDDataIndex()
		gdi.data.Unlock()

		// keep ctx cancel to be able to stop if necessary
		ctx2, cancel := context.WithCancel(ctx)
		gdi.cancel = cancel
		defer func() { gdi.cancel() }()

		gdi.updateUI()

		return gdi.run2(erow, args, ctx2, w)
	})

	return nil
}

func (gdi *GoDebugInstance) run2(erow *ERow, args []string, ctx context.Context, w io.Writer) error {
	cmd := godebug.NewCmd(args[1:], nil)
	defer cmd.Cleanup()

	cmd.Dir = erow.Info.Name()
	cmd.Stdout = w
	cmd.Stderr = w

	if err := cmd.Start(ctx); err != nil {
		return err
	}

	// output cmd pid
	fmt.Fprintf(w, "# pid %d\n", cmd.ServerCmd.Process.Pid)

	// handle client msgs loop (blocking)
	gdi.clientMsgsLoop(ctx, w, cmd)

	return cmd.Wait()
}

//----------

func (gdi *GoDebugInstance) clientMsgsLoop(ctx context.Context, w io.Writer, cmd *godebug.Cmd) {

	// request file positions before entering loop
	if err := cmd.RequestFileSetPositions(); err != nil {
		fmt.Fprint(w, err)
		return
	}

	// TODO: timeout to receive file set positions?

	// loop
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
			gdi.handleMsg(msg, w, cmd)
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

func (gdi *GoDebugInstance) handleMsg(msg interface{}, w io.Writer, cmd *godebug.Cmd) {
	if err := gdi.indexMsg(msg); err != nil {
		fmt.Fprint(w, err)
		return

	}

	// on receiving the filesdatamsg,  send a requeststart
	if _, ok := msg.(*debug.FilesDataMsg); ok {
		if err := cmd.RequestStart(); err != nil {
			err2 := errors.Wrap(err, "request start")
			fmt.Fprint(w, err2)
			return
		}
	}
}

func (gdi *GoDebugInstance) indexMsg(msg interface{}) error {
	gdi.data.Lock()
	defer gdi.data.Unlock()
	return gdi.data.dataIndex.IndexMsg(msg)
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

func (gdi *GoDebugInstance) updateUI2() {
	// update all infos (if necessary)
	for _, info := range gdi.ed.ERowInfos {
		gdi.updateInfoUI(info)
	}
}

func (gdi *GoDebugInstance) updateInfoUI(info *ERowInfo) {
	di := gdi.data.dataIndex
	clear := di == nil

	if clear {
		info.UpdateAnnotationsRowState(false)
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
	} else {
		findex, ok := di.FilesIndex[info.Name()]
		if !ok {
			info.UpdateAnnotationsRowState(false)
			return
		}

		// TODO
		//// check if edited
		//afd, ok := di.Afds[findex]
		//if !ok {
		//	return
		//}
		//if erow.Info.

		info.UpdateAnnotationsRowState(true)

		di := gdi.data.dataIndex
		fmsgs := &di.FileMsgs[findex]

		// setup lock/unlock each erow annotations
		for _, erow := range info.ERows {
			ta := erow.Row.TextArea
			if d, ok := ta.Drawer.(*drawer3.PosDrawer); ok {
				d.Annotations.Opt.EntriesMu.Lock()
				d.Annotations.Opt.EntriesMu.Unlock()
			}
		}

		// update annotations (safe after lock)
		fmsgs.updateAnnEntries(di.SelectedArrivalIndex)

		for _, erow := range info.ERows {
			ta := erow.Row.TextArea
			if d, ok := ta.Drawer.(*drawer3.PosDrawer); ok {
				d.Annotations.SetOn(true)
				d.Annotations.Opt.Select.Line = fmsgs.SelectedLine
				d.Annotations.Opt.Entries = fmsgs.AnnEntries
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
	FileMsgs             []GDFileMsgs               // file index -> file msgs
	GlobalArrivalIndex   int
	SelectedArrivalIndex int
}

func NewGDDataIndex() *GDDataIndex {
	di := &GDDataIndex{}
	di.FilesIndex = map[string]int{}
	return di
}

func (di *GDDataIndex) IndexMsg(msg interface{}) error {
	switch t := msg.(type) {
	case *debug.FilesDataMsg:
		di.Afds = t.Data
		// index filenames
		di.FilesIndex = map[string]int{}
		for _, afd := range di.Afds {
			di.FilesIndex[afd.Filename] = afd.FileIndex
		}
		// init index
		di.FileMsgs = make([]GDFileMsgs, len(di.Afds))
		for _, afd := range di.Afds {
			// check index
			if afd.FileIndex >= len(di.FileMsgs) {
				return fmt.Errorf("bad file index at init: %v len=%v", afd.FileIndex, len(di.FileMsgs))
			}
			fmsgs := &di.FileMsgs[afd.FileIndex]
			// init
			fmsgs.LineMsgs = make([]GDLineMsgs, afd.DebugLen)
			// init annotations entries
			fmsgs.AnnEntries = make([]*drawer3.Annotation, afd.DebugLen)
			fmsgs.AnnEntriesLMIndex = make([]int, afd.DebugLen)
		}
	case *debug.LineMsg:
		// check index
		l1 := len(di.FileMsgs)
		if t.FileIndex >= l1 {
			return fmt.Errorf("bad file index: %v len=%v", t.FileIndex, l1)
		}
		// check index
		l2 := len(di.FileMsgs[t.FileIndex].LineMsgs)
		if t.DebugIndex >= l2 {
			return fmt.Errorf("bad debug index: %v len=%v", t.DebugIndex, l2)
		}
		// line msg
		lm := &GDLineMsg{GlobalArrivalIndex: di.GlobalArrivalIndex, DLineMsg: t}
		// index msg
		w := &di.FileMsgs[t.FileIndex].LineMsgs[t.DebugIndex].Msgs
		*w = append(*w, lm)

		// auto update selected index if at last position
		if di.SelectedArrivalIndex == di.GlobalArrivalIndex-1 {
			di.SelectedArrivalIndex++
		}

		di.GlobalArrivalIndex++

		//// TODO: mark as having new data
		//di.FileMsgs[t.FileIndex].Updated = true

	default:
		return fmt.Errorf("unexpected msg: %T", msg)
	}
	return nil
}

//----------

type GDFileMsgs struct {
	//Updated bool // performance

	// all annotations received
	LineMsgs []GDLineMsgs

	// current annotation entries to be shown with a file
	AnnEntries        []*drawer3.Annotation
	AnnEntriesLMIndex []int // line messages index

	SelectedLine int
}

func (fmsgs *GDFileMsgs) updateAnnEntries(maxArrivalIndex int) {
	fmsgs.SelectedLine = -1
	for line, lm := range fmsgs.LineMsgs {
		k := sort.Search(len(lm.Msgs), func(i int) bool {
			u := lm.Msgs[i].GlobalArrivalIndex
			return u > maxArrivalIndex
		})
		// get less or equal then maxarrivalindex
		k--
		if k < 0 {
			fmsgs.AnnEntries[line] = nil
			if len(lm.Msgs) > 0 {
				fmsgs.AnnEntries[line] = lm.Msgs[0].emptyAnnotation()
			}
		} else {
			fmsgs.AnnEntries[line] = lm.Msgs[k].annotation()

			// selected line
			if lm.Msgs[k].GlobalArrivalIndex == maxArrivalIndex {
				fmsgs.SelectedLine = line
			}
		}

		// keep selected k to know the msg entry when coming from a click on an annotation
		fmsgs.AnnEntriesLMIndex[line] = k
	}
}

//----------

type GDLineMsgs struct {
	Msgs []*GDLineMsg
}

//----------

type GDLineMsg struct {
	GlobalArrivalIndex int
	DLineMsg           *debug.LineMsg
	itemBytes          []byte
	cachedAnn          *drawer3.Annotation
}

func (lmsg *GDLineMsg) build() *drawer3.Annotation {
	if lmsg.cachedAnn == nil {
		lmsg.cachedAnn = &drawer3.Annotation{Offset: lmsg.DLineMsg.Offset}
	}
	return lmsg.cachedAnn
}

func (lmsg *GDLineMsg) annotation() *drawer3.Annotation {
	ann := lmsg.build()

	// stringify item
	if lmsg.itemBytes == nil {
		lmsg.itemBytes = []byte(godebug.StringifyItem(lmsg.DLineMsg.Item))
	}
	ann.Bytes = lmsg.itemBytes

	return ann
}

func (lmsg *GDLineMsg) emptyAnnotation() *drawer3.Annotation {
	ann := lmsg.build()
	ann.Bytes = []byte(" ")
	return ann
}
