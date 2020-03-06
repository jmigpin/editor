package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jmigpin/editor/core/toolbarparser"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/uiutil/event"
)

//----------

type ERow struct {
	Ed     *Editor
	Row    *ui.Row
	Info   *ERowInfo
	Exec   *ERowExec
	TbData toolbarparser.Data

	highlightDuplicates bool

	parsingToolbar bool // protect against write loop

	termFilter bool

	ctx       context.Context // erow general context
	cancelCtx context.CancelFunc

	cmd struct {
		sync.Mutex
		cancelInternalCmd context.CancelFunc
		cancelContentCmd  context.CancelFunc
	}
}

//----------

func NewERow(ed *Editor, info *ERowInfo, rowPos *ui.RowPos) *ERow {
	// create row
	row := rowPos.Column.NewRowBefore(rowPos.NextRow)

	erow := &ERow{Ed: ed, Row: row, Info: info}
	erow.Exec = NewERowExec(erow)
	ctx0 := context.Background() // TODO: editor ctx
	erow.ctx, erow.cancelCtx = context.WithCancel(ctx0)

	erow.setupTextAreaSyntaxHighlight()

	erow.initHandlers()

	// init name; any string (len>0) will be replaced by the encoded name
	erow.updateToolbarNameEncoding2("_")

	// editor events
	ev := &PostNewERowEEvent{ERow: erow}
	erow.Ed.EEvents.emit(PostNewERowEEventId, ev)

	return erow
}

//----------

func (erow *ERow) initHandlers() {
	row := erow.Row

	// register with the editor
	erow.Ed.SetERowInfo(erow.Info.Name(), erow.Info)
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

	// toolbar on prewrite
	row.Toolbar.RWEvReg.Add(iorw.RWEvIdPreWrite, func(ev0 interface{}) {
		ev := ev0.(*iorw.RWEvPreWrite)
		if err := erow.validateToolbarPreWrite(ev); err != nil {
			ev.ReplyErr = err
		}
	})
	// toolbar cmds
	row.Toolbar.EvReg.Add(ui.TextAreaCmdEventId, func(ev0 interface{}) {
		InternalCmdFromRowTb(erow)
	})
	// textarea on write
	row.TextArea.RWEvReg.Add(iorw.RWEvIdWrite, func(ev0 interface{}) {
		ev := ev0.(*iorw.RWEvWrite)
		erow.Info.HandleRWEvWrite(erow, ev)
	})
	// textarea content cmds
	row.TextArea.EvReg.Add(ui.TextAreaCmdEventId, func(ev0 interface{}) {
		ev := ev0.(*ui.TextAreaCmdEvent)
		ContentCmdFromTextArea(erow, ev.Index)
	})
	// textarea select annotation
	row.TextArea.EvReg.Add(ui.TextAreaSelectAnnotationEventId, func(ev interface{}) {
		ev2 := ev.(*ui.TextAreaSelectAnnotationEvent)
		erow.Ed.GoDebug.SelectERowAnnotation(erow, ev2)
	})
	// textarea inlinecomplete
	row.TextArea.EvReg.Add(ui.TextAreaInlineCompleteEventId, func(ev0 interface{}) {
		ev := ev0.(*ui.TextAreaInlineCompleteEvent)
		handled := erow.Ed.InlineComplete.Complete(erow, ev)
		// Allow the input event (`tab` key press) to function normally if the inlinecomplete is not being handled (ex: no lsproto server is registered for this filename extension)
		ev.ReplyHandled = event.Handled(handled)
	})
	// key shortcuts
	row.EvReg.Add(ui.RowInputEventId, func(ev0 interface{}) {
		erow.Ed.InlineComplete.CancelOnCursorChange()

		ev := ev0.(*ui.RowInputEvent)
		switch evt := ev.Event.(type) {
		case *event.KeyDown:
			// activate row
			erow.Info.UpdateActiveRowState(erow)
			// shortcuts
			mods := evt.Mods.ClearLocks()
			switch {
			case mods.Is(event.ModCtrl) && evt.KeySym == event.KSymS:
				if err := erow.Info.SaveFile(); err != nil {
					erow.Ed.Error(err)
				}
			case mods.Is(event.ModCtrl) && evt.KeySym == event.KSymF:
				FindShortcut(erow)
			}
		case *event.MouseDown:
			erow.Info.UpdateActiveRowState(erow)
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
		// editor events
		ev := &PreRowCloseEEvent{ERow: erow}
		erow.Ed.EEvents.emit(PreRowCloseEEventId, ev)

		// cancel general context
		erow.cancelCtx()

		// ensure execution (if any) is stopped
		erow.Exec.Stop()

		// unregister from editor
		erow.Info.RemoveERow(erow)
		if len(erow.Info.ERows) == 0 {
			erow.Ed.DeleteERowInfo(erow.Info.Name())
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

func (erow *ERow) encodedName() string {
	return erow.Ed.HomeVars.Encode(erow.Info.Name())
}

//----------

func (erow *ERow) validateToolbarPreWrite(ev *iorw.RWEvPreWrite) error {
	// current content (pre write) copy
	b, err := iorw.ReadFullCopy(erow.Row.Toolbar.RW())
	if err != nil {
		return err
	}

	// simulate the write // TODO: how to guarantee the simulation is accurate and no rw filter exists.
	rw := iorw.NewBytesReadWriter(b)
	if err := rw.Overwrite(ev.Index, ev.N, ev.P); err != nil {
		return err
	}
	b2, err := iorw.ReadFullFast(rw)
	if err != nil {
		return err
	}
	tbStr2 := string(b2)

	// simulation name
	data := toolbarparser.Parse(tbStr2)
	arg0, ok := data.Part0Arg0()
	if !ok {
		return fmt.Errorf("unable to get toolbar name")
	}
	simName := arg0.UnquotedStr()

	// expected name
	nameEnc := erow.encodedName()

	if simName != nameEnc {
		return fmt.Errorf("can't change toolbar name: %q -> %q", nameEnc, simName)
	}

	// valid data
	erow.TbData = *data
	erow.parseToolbarVars()

	return nil
}

//----------

func (erow *ERow) UpdateToolbarNameEncoding() {
	str := erow.Row.Toolbar.Str()
	erow.updateToolbarNameEncoding2(str)
}

func (erow *ERow) updateToolbarNameEncoding2(str string) {
	data := toolbarparser.Parse(str)
	arg0, ok := data.Part0Arg0()
	if !ok {
		return
	}

	// replace part0 arg0 with encoded name
	ename := erow.encodedName()
	str2 := ename + str[arg0.End:]
	if str2 != str {
		erow.Row.Toolbar.SetStrClearHistory(str2)
	}
}

//----------

func (erow *ERow) ToolbarSetStrAfterNameClearHistory(s string) {
	arg0, ok := erow.TbData.Part0Arg0()
	if !ok {
		return
	}
	str := erow.Row.Toolbar.Str()[:arg0.End] + s
	erow.Row.Toolbar.SetStrClearHistory(str)
}

//----------

func (erow *ERow) parseToolbarVars() {
	vmap := toolbarparser.ParseVars(&erow.TbData)

	// $font
	clear := true
	if v, ok := vmap["$font"]; ok {
		err := erow.setVarFontTheme(v)
		if err == nil {
			clear = false
		}
	}
	if clear {
		erow.Row.TextArea.SetThemeFont(nil)
	}

	// $termFilter
	erow.termFilter = false
	if v, ok := vmap["$termFilter"]; ok {
		if v == "" || strings.ToLower(v) == "true" {
			erow.termFilter = true
		}
	}
}

func (erow *ERow) setVarFontTheme(s string) error {
	tf, err := ui.ThemeFont(s)
	if err != nil {
		return err
	}
	erow.Row.TextArea.SetThemeFont(tf)
	return nil
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

// Deprecated: use textAreaAppendBytesUIWriter().
func (erow *ERow) TextAreaAppendBytesAsync(p []byte) <-chan struct{} {
	comm := make(chan struct{})
	erow.Ed.UI.RunOnUIGoRoutine(func() {
		erow.TextAreaAppendBytes(p)
		close(comm)
	})
	return comm
}

func (erow *ERow) TextAreaAppendBytes(p []byte) {
	ta := erow.Row.TextArea
	if err := ta.AppendBytesClearHistory(p); err != nil {
		erow.Ed.Error(err)
	}
}

// UI Safe. Writer will block until it has queued in the UI goroutine (the actual apppend bytes is done later in the UI goroutine, avoiding possible UI locks).
func (erow *ERow) TextAreaAppendBytesUIWriter() io.Writer {
	return iout.FnWriter(func(b []byte) (int, error) {
		// can't sync.wait since it could lock the caller if inside a uigoroutine

		// make copy since the data will be used async in uigoroutine
		b2 := make([]byte, len(b))
		copy(b2, b)

		erow.Ed.UI.RunOnUIGoRoutine(func() {
			err := erow.Row.TextArea.AppendBytesClearHistory(b2)
			if err != nil {
				erow.Ed.Error(err)
			}
		})

		return len(b2), nil
	})
}

//----------

// UI Safe. Caller is responsible for closing the writer at the end.
func (erow *ERow) TextAreaWriter() io.WriteCloser {
	// terminal filter (escape sequences)
	// avoid data race: assign before goroutine
	termFilter := erow.termFilter && erow.Info.IsDir()

	prc, pwc := io.Pipe()
	var copyLoop sync.WaitGroup
	copyLoop.Add(1)
	go func() {
		defer copyLoop.Done()
		var r io.Reader = prc
		if termFilter {
			r = NewTerminalFilter(erow, r)
		}
		w := erow.TextAreaAppendBytesUIWriter()
		if _, err := io.Copy(w, r); err != nil {
			prc.Close()
		}
	}()
	// wrap pwc for performance (buffered) with output visible (auto-flush)
	abw := iout.NewAutoBufWriter(pwc)
	//return abw

	// wait for the copy loop to finish on close
	type waitWriteCloser struct {
		io.Writer
		io.Closer
	}
	closer := iout.FnCloser(func() error {
		err := abw.Close()
		copyLoop.Wait()
		return err
	})
	return &waitWriteCloser{Writer: abw, Closer: closer}
}

//----------

// UI Safe
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
	// Commented: don't flicker row positions
	//erow.Row.EnsureTextAreaMinimumHeight()
	erow.Row.EnsureOneToolbarLineYVisible()

	erow.Row.TextArea.MakeRangeVisible(index, len)
	erow.Row.TextArea.FlashIndexLen(index, len)

	// flash toolbar as last resort if less visible
	ta := erow.Row.TextArea
	lh := ta.LineHeight()
	min := int(float64(lh) * 1.5)
	if ta.Bounds.Dy() < min {
		erow.Flash()
	}
}

//----------

func (erow *ERow) setupTextAreaSyntaxHighlight() {
	ta := erow.Row.TextArea

	// util funcs
	setComments := func(a ...interface{}) {
		ta.EnableSyntaxHighlight(true) // ensure syntax highlight is on
		ta.SetCommentStrings(a...)
	}

	// ignore "." on files starting with "."
	name := filepath.Base(erow.Info.Name())
	if len(name) >= 1 && name[0] == '.' {
		name = name[1:]
	}

	// specific names
	switch name {
	case "bashrc":
		setComments("#")
		return
	case "go.mod":
		setComments("//")
		return
	}

	// name extension
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".sh",
		".conf", ".list",
		".py", // python
		".pl": // perl
		setComments("#")
	case ".go",
		".c", ".h",
		".cpp", ".hpp", ".cxx", ".hxx", // c++
		".java",
		".js": // javascript
		setComments("//", [2]string{"/*", "*/"})
	case ".ledger":
		setComments(";", "//")
	case ".pro": // prolog
		setComments("%", [2]string{"/*", "*/"})
	case ".html", ".xml", ".svg":
		setComments([2]string{"<!--", "-->"})
	case ".s", ".asm": // assembly
		setComments("//")
	case ".json": // no comments to setup
		ta.EnableSyntaxHighlight(true)
	case ".txt":
		setComments("#") // useful (but not correct)
	case "": // no file extension (includes directories and special rows)
		setComments("#") // useful (but not correct)
	default: // all other file extensions
		ta.EnableSyntaxHighlight(true)
	}
}

//----------

func (erow *ERow) newContentCmdCtx() (context.Context, context.CancelFunc) {
	erow.cmd.Lock()
	defer erow.cmd.Unlock()
	erow.cancelContentCmd2()
	ctx, cancel := context.WithCancel(erow.ctx)
	erow.cmd.cancelContentCmd = cancel
	return ctx, cancel
}
func (erow *ERow) CancelContentCmd() {
	erow.cmd.Lock()
	defer erow.cmd.Unlock()
	erow.cancelContentCmd2()
}
func (erow *ERow) cancelContentCmd2() {
	if erow.cmd.cancelContentCmd != nil {
		erow.cmd.cancelContentCmd()
	}
}

//----------

func (erow *ERow) newInternalCmdCtx() (context.Context, context.CancelFunc) {
	erow.cmd.Lock()
	defer erow.cmd.Unlock()
	erow.cancelInternalCmd2()
	ctx, cancel := context.WithCancel(erow.ctx)
	erow.cmd.cancelInternalCmd = cancel
	return ctx, cancel
}

func (erow *ERow) CancelInternalCmd() {
	erow.cmd.Lock()
	defer erow.cmd.Unlock()
	erow.cancelInternalCmd2()
}
func (erow *ERow) cancelInternalCmd2() {
	if erow.cmd.cancelInternalCmd != nil {
		erow.cmd.cancelInternalCmd()
	}
}
