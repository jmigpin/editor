package core

import (
	"bufio"
	"errors"
	"io"
	"path/filepath"
	"sync"
	"time"

	"github.com/jmigpin/editor/core/toolbarparser"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/drawutil/drawer3"
	"github.com/jmigpin/editor/util/uiutil/event"
)

//----------

type ERow struct {
	Ed     *Editor
	Row    *ui.Row
	Info   *ERowInfo
	Exec   ERowExec
	TbData toolbarparser.Data

	highlightDuplicates           bool
	disableTextAreaSetStrCallback bool
}

//----------

func NewERow(ed *Editor, info *ERowInfo, rowPos *ui.RowPos) *ERow {
	// create row
	row := rowPos.Column.NewRowBefore(rowPos.NextRow)

	erow := &ERow{Ed: ed, Row: row, Info: info}
	erow.Exec.erow = erow

	// TODO: join with updateToolbarPart0
	s2 := ed.HomeVars.Encode(erow.Info.Name())
	erow.Row.Toolbar.SetStrClearHistory(s2)

	erow.initHandlers()
	erow.parseToolbar() // after handlers are set
	erow.setupTextAreaCommentString()

	return erow
}

//----------

func (erow *ERow) initHandlers() {
	row := erow.Row

	// register with the editor
	erow.Ed.ERowInfos[erow.Info.Name()] = erow.Info
	erow.Info.AddERow(erow)

	// update row state
	erow.Info.UpdateDuplicateRowState()
	erow.Info.UpdateDuplicateHighlightRowState()
	erow.Info.UpdateExistsRowState()
	erow.Info.UpdateFsDifferRowState()

	// register with watcher
	if !erow.Info.IsSpecial() && len(erow.Info.ERows) == 1 {
		erow.Ed.Watcher.Add(erow.Info.Name())
	}

	// toolbar set str
	row.Toolbar.EvReg.Add(ui.TextAreaSetStrEventId, func(ev0 interface{}) {
		erow.parseToolbar()
	})
	// toolbar cmds
	row.Toolbar.EvReg.Add(ui.TextAreaCmdEventId, func(ev0 interface{}) {
		RowToolbarCmd(erow)
	})
	// textarea set str
	row.TextArea.EvReg.Add(ui.TextAreaSetStrEventId, func(ev0 interface{}) {
		//ev := ev0.(*ui.TextAreaSetStrEvent)

		// TODO: setup this somewhere else
		// update xbar
		if d, ok := erow.Row.TextArea.Drawer.(*drawer3.PosDrawer); ok {
			erow.Row.EnableTextAreaXBar(!d.WrapLine.On())
		}

		if erow.disableTextAreaSetStrCallback {
			return
		}

		erow.Info.SetRowsStrFromMaster(erow)

		// TODO: update godebug annotations if hash doesn't match
		//GoDebugCheckTextAreaContent(erow)
	})
	// textarea content cmds
	row.TextArea.EvReg.Add(ui.TextAreaCmdEventId, func(ev0 interface{}) {
		ev := ev0.(*ui.TextAreaCmdEvent)
		RunContentCmds(erow, ev.Index)
	})
	// textarea select annotation
	row.TextArea.EvReg.Add(ui.TextAreaSelectAnnotationEventId, func(ev0 interface{}) {
		ev := ev0.(*ui.TextAreaSelectAnnotationEvent)
		GoDebugSelectAnnotation(erow, ev.AnnotationIndex, ev.Offset, ev.Type)
	})
	// key shortcuts
	row.EvReg.Add(ui.RowInputEventId, func(ev0 interface{}) {
		ev := ev0.(*ui.RowInputEvent)
		switch evt := ev.Event.(type) {
		case *event.KeyDown:
			mods := evt.Mods.ClearLocks()
			switch {
			case mods.Is(event.ModCtrl) && evt.LowerRune() == 's':
				if err := erow.Info.SaveFile(); err != nil {
					erow.Ed.Error(err)
				}
			case mods.Is(event.ModCtrl) && evt.LowerRune() == 'f':
				FindShortcut(erow)
			}
		case *event.MouseEnter:
			erow.highlightDuplicates = true
			erow.Info.UpdateDuplicateHighlightRowState()
		case *event.MouseLeave:
			erow.highlightDuplicates = false
			erow.Info.UpdateDuplicateHighlightRowState()
		}
	})
	// close
	row.EvReg.Add(ui.RowCloseEventId, func(ev0 interface{}) {
		// ensure execution (if any) is stopped
		erow.Exec.Stop()

		// unregister from editor
		erow.Info.RemoveERow(erow)
		if len(erow.Info.ERows) == 0 {
			delete(erow.Ed.ERowInfos, erow.Info.Name())
		}

		// update row state
		erow.Info.UpdateDuplicateRowState()
		erow.Info.UpdateDuplicateHighlightRowState()

		// unregister with watcher
		if !erow.Info.IsSpecial() && len(erow.Info.ERows) == 0 {
			erow.Ed.Watcher.Remove(erow.Info.Name())
		}

		// add to reopener to allow to reopen later if needed
		if !erow.Info.IsSpecial() {
			erow.Ed.RowReopener.Add(row)
		}
	})
}

//----------

func (erow *ERow) parseToolbar() {
	str := erow.Row.Toolbar.Str()

	data := toolbarparser.Parse(str)

	// don't allow toolbar edit of the name
	ename := erow.Ed.HomeVars.Encode(erow.Info.Name())
	arg0, ok := data.Part0Arg0()
	if !ok {
		return
	}
	ename2 := arg0.UnquotedStr()
	if ename2 != ename {
		erow.Row.Toolbar.TextHistory.Undo()
		erow.Row.Toolbar.TextHistory.ClearForward()
		erow.Ed.Errorf("can't change toolbar name")
		return
	}

	erow.TbData = *data
}

//----------

func (erow *ERow) updateToolbarPart0() {
	str := erow.Row.Toolbar.Str()
	data := toolbarparser.Parse(str)
	arg0, ok := data.Part0Arg0()
	if !ok {
		return
	}

	// replace part0 arg0 with encoded name
	ename := erow.Ed.HomeVars.Encode(erow.Info.Name())
	str2 := ename + str[arg0.End:]
	if str2 != str {
		erow.Row.Toolbar.SetStrClearHistory(str2)
	}
}

//----------

func (erow *ERow) Reload() {
	if err := erow.reload(); err != nil {
		erow.Ed.Error(err)
	}
}

func (erow *ERow) reload() error {
	switch {
	case erow.Info.IsDir():
		return erow.Info.ReloadDir(erow)
	case erow.Info.IsFileButNotDir():
		return erow.Info.ReloadFile()
	default:
		return errors.New("unexpected type to reload")
	}
}

//----------

func (erow *ERow) ToolbarSetStrAfterNameClearHistory(s string) {
	arg, ok := erow.TbData.Part0Arg0()
	if !ok {
		return
	}
	i := arg.End
	str := erow.Row.Toolbar.Str()[:i] + s
	erow.Row.Toolbar.SetStrClearHistory(str)
}

//----------

func (erow *ERow) TextAreaAppendAsync(str string) <-chan struct{} {
	comm := make(chan struct{})
	erow.Ed.UI.RunOnUIGoRoutine(func() {
		erow.textAreaAppend(str)
		close(comm)
	})
	return comm
}

func (erow *ERow) textAreaAppend(str string) {
	// TODO: unlimited output? some xterms have more or less 64k limit. Bigger limits will slow down the ui since it will be calculating the new string content. This will be improved once the textarea drawer supports append/cutTop operations.

	maxSize := 64 * 1024

	ta := erow.Row.TextArea
	if err := ta.AppendStrClearHistory(str, maxSize); err != nil {
		erow.Ed.Error(err)
	}
}

//----------

// Caller is responsible for closing the writer at the end.
func (erow *ERow) TextAreaWriter() io.WriteCloser {
	pr, pw := io.Pipe()
	go func() {
		erow.readLoopToTextArea(pr)
	}()
	return NewBufWriter(pw)
}

func (erow *ERow) readLoopToTextArea(reader io.Reader) {
	var buf [4 * 1024]byte
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

//----------

func (erow *ERow) Flash() {
	p, ok := erow.TbData.PartAtIndex(0)
	if ok {
		if len(p.Args) > 0 {
			a := p.Args[0]
			erow.Row.Toolbar.FlashIndexLen(a.Pos, a.End-a.Pos)
		}
	}
}

//----------

func (erow *ERow) MakeIndexVisibleAndFlash(index int) {
	erow.MakeRangeVisibleAndFlash(index, 0)
}
func (erow *ERow) MakeRangeVisibleAndFlash(index int, len int) {
	erow.Row.EnsureTextAreaMinimumHeight()
	erow.Row.TextArea.MakeRangeVisible(index, len)
	erow.Row.TextArea.FlashIndexLen(index, len)

	// flash toolbar as last resort
	//if !erow.Row.TextArea.IsRangeVisible(index, len) {
	b := &erow.Row.TextArea.Bounds
	if b.Dx() < 10 || b.Dy() < 10 { // TODO: use dpi instead of fixed pixels
		erow.Flash()
	}
}

//----------

func (erow *ERow) setupTextAreaCommentString() {
	ta := erow.Row.TextArea
	switch filepath.Ext(erow.Info.Name()) {
	default:
		fallthrough
	case "", ".sh", ".conf", ".list", ".txt":
		ta.SetCommentStrings("#", [2]string{})
	case ".go", ".c", ".cpp", ".h", ".hpp":
		ta.SetCommentStrings("//", [2]string{"/*", "*/"})
	}
}

//----------

// Used by erow.TextAreaWriter. Safe to use concurrently.
// Auto flushes after x time if the buffer doesn't get filled.
type BufWriter struct {
	mu    sync.Mutex
	buf   *bufio.Writer
	wc    io.WriteCloser
	timer *time.Timer
}

func NewBufWriter(wc io.WriteCloser) *BufWriter {
	buf := bufio.NewWriter(wc)
	return &BufWriter{buf: buf, wc: wc}
}

// Implements io.Closer
func (bw *BufWriter) Close() error {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	bw.clearTimer()
	bw.buf.Flush()
	return bw.wc.Close()
}

// Implements io.Writer
func (bw *BufWriter) Write(p []byte) (int, error) {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	defer bw.autoFlush() // deferred to run after the write
	return bw.buf.Write(p)
}

func (bw *BufWriter) autoFlush() {
	if bw.buf.Buffered() == 0 {
		bw.clearTimer()
		return
	}
	if bw.timer == nil {
		bw.timer = time.AfterFunc(50*time.Millisecond, bw.flushTime)
	}
}
func (bw *BufWriter) flushTime() {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	bw.buf.Flush()
	bw.clearTimer()
}

func (bw *BufWriter) clearTimer() {
	if bw.timer != nil {
		bw.timer.Stop()
		bw.timer = nil
	}
}
