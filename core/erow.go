package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/jmigpin/editor/core/toolbarparser"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/drawutil"
	"github.com/jmigpin/editor/util/drawutil/drawer4"
	"github.com/jmigpin/editor/util/fontutil"
	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/uiutil/event"
)

type ERow struct {
	Ed     *Editor
	Row    *ui.Row
	Info   *ERowInfo
	Exec   *ERowExec
	TbData toolbarparser.Data

	highlightDuplicates bool

	terminalOpt    terminalOpt
	scrollDownMode string

	ctx       context.Context // erow general context
	cancelCtx context.CancelFunc

	cmd struct {
		sync.Mutex
		cancelInternalCmd context.CancelFunc
		cancelContentCmd  context.CancelFunc
	}
}

//----------

func NewLoadedERow(info *ERowInfo, rowPos *ui.RowPos) (*ERow, error) {
	switch {
	case info.IsSpecial():
		return newLoadedSpecialERow(info, rowPos)
	case info.IsDir():
		return newLoadedDirERow(info, rowPos)
	case info.IsFileButNotDir():
		return newLoadedFileERow(info, rowPos)
	default:
		err := fmt.Errorf("unable to open erow: %v", info.name)
		if info.fiErr != nil {
			err = fmt.Errorf("%v: %w", err, info.fiErr)
		}
		return nil, err
	}
}

// Allows creating rows in place even if a file/dir doesn't exist anymore (ex: show non-existent files rows in a saved session).
func NewLoadedERowOrNewBasic(info *ERowInfo, rowPos *ui.RowPos) *ERow {
	erow, err := NewLoadedERow(info, rowPos)
	if err != nil {
		return NewBasicERow(info, rowPos)
	}
	return erow
}

//----------

func ExistingERowOrNewLoaded(ed *Editor, name string) (_ *ERow, isNew bool, _ error) {
	info := ed.ReadERowInfo(name)
	if erow0, ok := info.FirstERow(); ok {
		return erow0, false, nil
	}
	rowPos := ed.GoodRowPos()
	erow, err := NewLoadedERow(info, rowPos)
	if err != nil {
		return nil, false, err
	}
	return erow, true, nil
}

// Used for ex. in: +messages, +sessions.
func ExistingERowOrNewBasic(ed *Editor, name string) (_ *ERow, isNew bool) {

	info := ed.ReadERowInfo(name)
	if erow0, ok := info.FirstERow(); ok {
		return erow0, false
	}
	rowPos := ed.GoodRowPos()
	erow := NewBasicERow(info, rowPos)
	return erow, true
}

//----------

func NewBasicERow(info *ERowInfo, rowPos *ui.RowPos) *ERow {
	erow := &ERow{}
	erow.init(info, rowPos)
	return erow
}

func (erow *ERow) init(info *ERowInfo, rowPos *ui.RowPos) {
	erow.Ed = info.Ed
	erow.Info = info
	erow.Row = rowPos.Column.NewRowBefore(rowPos.NextRow)
	erow.Exec = NewERowExec(erow)

	ctx0 := context.Background() // TODO: editor ctx
	erow.ctx, erow.cancelCtx = context.WithCancel(ctx0)

	detectSetupSyntaxHighlight(erow)
	erow.initHandlers()

	erow.updateToolbarNameEncoding2("")

	// editor events
	ev := &PostNewERowEEvent{ERow: erow}
	erow.Ed.EEvents.emit(PostNewERowEEventId, ev)
}

//----------

func newLoadedSpecialERow(info *ERowInfo, rowPos *ui.RowPos) (*ERow, error) {
	// there can be only one instance of a special row
	if len(info.ERows) > 0 {
		return nil, fmt.Errorf("special row already exists: %v", info.Name())

	}
	erow := NewBasicERow(info, rowPos)
	// load
	switch {
	case info.Name() == "+Sessions":
		ListSessions(erow.Ed)
	}
	return erow, nil
}

func newLoadedDirERow(info *ERowInfo, rowPos *ui.RowPos) (*ERow, error) {
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory")
	}
	erow := NewBasicERow(info, rowPos)
	// load
	ListDirERow(erow, erow.Info.Name(), false, true)
	return erow, nil
}

func newLoadedFileERow(info *ERowInfo, rowPos *ui.RowPos) (*ERow, error) {
	// read content from existing row
	if erow0, ok := info.FirstERow(); ok {
		// create erow first to get it updated
		erow := NewBasicERow(info, rowPos)
		// update the new erow with content
		info.setRWFromMaster(erow0)
		return erow, nil
	}

	// load
	b, err := info.readFsFile()
	if err != nil {
		return nil, err
	}

	// update data
	info.setSavedHash(info.fileData.fs.hash, len(b))

	// new erow (no other rows exist)
	erow := NewBasicERow(info, rowPos)
	erow.Row.TextArea.SetBytesClearHistory(b)

	return erow, nil
}

//----------

func (erow *ERow) Reload() error {
	switch {
	case erow.Info.IsSpecial() && erow.Info.Name() == "+Sessions":
		ListSessions(erow.Ed)
		return nil
	case erow.Info.IsDir():
		ListDirERow(erow, erow.Info.Name(), false, true)
		return nil
	case erow.Info.IsFileButNotDir():
		return erow.Info.ReloadFile()
	case erow.Info.FileInfoErr() != nil:
		return erow.Info.FileInfoErr()
	default:
		return errors.New("unexpected type to reload")
	}
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
	row.Toolbar.RWEvReg.Add(iorw.RWEvIdPreWrite, func(ev0 any) {
		ev := ev0.(*iorw.RWEvPreWrite)
		if err := erow.validateToolbarPreWrite(ev); err != nil {
			ev.ReplyErr = err
		}
	})
	// toolbar cmds
	row.Toolbar.EvReg.Add(ui.TextAreaCmdEventId, func(ev0 any) {
		InternalOrExternalCmdFromRowTb(erow)
	})
	// textarea on write
	row.TextArea.RWEvReg.Add(iorw.RWEvIdWrite2, func(ev0 any) {
		ev := ev0.(*iorw.RWEvWrite2)
		erow.Info.HandleRWEvWrite2(erow, ev)
	})
	// textarea content cmds
	row.TextArea.EvReg.Add(ui.TextAreaCmdEventId, func(ev0 any) {
		ev := ev0.(*ui.TextAreaCmdEvent)
		ContentCmdFromTextArea(erow, ev.Index)
	})
	// textarea select annotation
	row.TextArea.EvReg.Add(ui.TextAreaSelectAnnotationEventId, func(ev any) {
		ev2 := ev.(*ui.TextAreaSelectAnnotationEvent)
		erow.Ed.GoDebug.SelectERowAnnotation(erow, ev2)
	})
	// textarea inlinecomplete
	row.TextArea.EvReg.Add(ui.TextAreaInlineCompleteEventId, func(ev0 any) {
		ev := ev0.(*ui.TextAreaInlineCompleteEvent)
		handled := erow.Ed.AnnotationsHandled(erow, ev)
		ev.ReplyHandled = event.Handled(handled)
	})
	// key shortcuts
	row.EvReg.Add(ui.RowInputEventId, func(ev0 any) {
		ev := ev0.(*ui.RowInputEvent)

		switch ev.Event.(type) {
		case *event.KeyDown, *event.MouseDown:
			erow.Ed.AnnotationsOnMouseKeyDown()
		}

		switch evt := ev.Event.(type) {
		case *event.KeyDown:
			// activate row
			erow.Info.UpdateActiveRowState(erow)
			// shortcuts
			mods := evt.Mods.ClearLocks()
			switch {
			case mods.Is(event.ModCtrl) && evt.KeySym == event.KSymS:
				erow.SaveFileBusyCursor()
			case mods.Is(event.ModCtrl) && evt.KeySym == event.KSymF:
				AddFindShortcut(erow)
			case mods.Is(event.ModCtrl|event.ModShift) && evt.KeySym == event.KSymF:
				// internal cmd
				str := "FillAssist -template"
				data := toolbarparser.Parse(str)
				part, _ := data.PartAtIndex(0)
				internalOrExternalCmd(erow.Ed, part, erow)
			case mods.Is(event.ModCtrl) && evt.KeySym == event.KSymH:
				AddReplaceShortcut(erow)
			case mods.Is(event.ModCtrl) && evt.KeySym == event.KSymN:
				AddNewFileShortcut(erow)
			case mods.Is(event.ModCtrl) && evt.KeySym == event.KSymR:
				AddReloadShortcut(erow)
			case mods.Is(event.ModCtrl) && evt.KeySym == event.KSymW:
				row.Close()
			case evt.KeySym == event.KSymEscape:
				erow.Exec.Stop()
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
	row.EvReg.Add(ui.RowCloseEventId, func(ev0 any) {
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

	// simulate the write
	// TODO: how to guarantee the simulation is accurate and no rw filter exists.
	rw := iorw.NewBytesReadWriterAt(b)
	if err := rw.OverwriteAt(ev.Index, ev.N, ev.P); err != nil {
		return err
	}
	b2, err := iorw.ReadFastFull(rw)
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
	simName := arg0.UnquotedString()

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
	arg0End := 0
	data := toolbarparser.Parse(str)
	arg0, ok := data.Part0Arg0()
	if ok {
		arg0End = arg0.End()
	}

	// replace part0 arg0 with encoded name
	ename := erow.encodedName()
	str2 := ename + str[arg0End:]
	if str2 != str {
		erow.Row.Toolbar.SetStrClearHistory(str2)
	}
}

func (erow *ERow) ToolbarSetStrAfterNameClearHistory(s string) {
	arg0, ok := erow.TbData.Part0Arg0()
	if !ok {
		return
	}
	str := erow.Row.Toolbar.Str()[:arg0.End()] + s
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
		erow.Row.TextArea.SetThemeFontFace(nil)
	}

	// $terminal
	erow.terminalOpt = terminalOpt{}
	if erow.Info.IsDir() {
		// Deprecated: use $terminal
		if _, ok := vmap["$termFilter"]; ok {
			erow.terminalOpt.filter = true
		}

		if v, ok := vmap["$terminal"]; ok {
			u := strings.Split(v, ",")
			for _, k := range u {
				switch k {
				case "f":
					erow.terminalOpt.filter = true
				case "k":
					erow.terminalOpt.keyEvents = true
				}
			}
		}
	}

	// $scrollMode: "auto", otherwise is "manual"/"off"
	erow.scrollDownMode = ""
	if v, ok := vmap["$scrollMode"]; ok {
		erow.scrollDownMode = v
	}
}

func (erow *ERow) setVarFontTheme(s string) error {
	w := strings.SplitN(s, ",", 2)
	name := strings.TrimSpace(w[0])

	// font size arg
	size := float64(0)
	if len(w) > 1 {
		v, err := strconv.ParseFloat(w[1], 64)
		if err != nil {
			// commented: ignore error
			//return err
		} else {
			size = v
		}
	}

	ta := erow.Row.TextArea

	// use parent node face options (might contain dpi)
	face := ta.Parent.TreeThemeFontFace()
	fopts2 := face.Opts // copy

	// start accepting only at 2, allows typing 1x without affecting the rendering
	if size >= 2 {
		fopts2.SetSize(size)
		// change only size if name not provided
		if len(name) == 0 {
			ff := face.Font.FontFace(fopts2)
			ta.SetThemeFontFace(ff)
			return nil
		}
	}

	ff := (*fontutil.FontFace)(nil)
	if len(name) == 0 { // change only size if name not provided
		ff = face.Font.FontFace(fopts2)
	} else {
		ff2, err := ui.ThemeFontFace2(name, fopts2)
		if err != nil {
			return err
		}
		ff = ff2
	}

	ta.SetThemeFontFace(ff)
	return nil
}

//----------

// Not UI safe.
func (erow *ERow) AppendBytesClearHistory(p []byte) {
	if err := erow.AppendBytesClearHistory2(p); err != nil {
		erow.Ed.Error(err)
	}
}
func (erow *ERow) AppendBytesClearHistory2(p []byte) error {
	ta := erow.Row.TextArea

	scrollDownModeAuto := false
	if erow.scrollDownMode == "auto" {
		if ta.IndexVisible(ta.RW().Max()) {
			scrollDownModeAuto = true
		}
	}

	if err := ta.AppendBytesClearHistory(p); err != nil {
		return err
	}

	if scrollDownModeAuto {
		// TODO: better drawutil.RAlignBottom? issues with sometimes losing the bottom hook
		ta.MakeRangeVisible2(ta.RW().Max(), 0, drawutil.RAlignKeepOrBottom)
	}
	return nil
}

//----------

func (erow *ERow) TextAreaReadWriteCloser() io.ReadWriteCloser {
	if erow.terminalOpt.On() {
		return NewTerminalFilter(erow)
	}

	// synced writer to slow down memory usage
	w := iout.FnWriter(func(b []byte) (int, error) {
		var err error
		erow.Ed.UI.WaitRunOnUIGoRoutine(func() {
			err = erow.AppendBytesClearHistory2(b)
		})
		return len(b), err
	})

	// buffered for performance, which needs timed output (auto-flush)
	wc := iout.NewAutoBufWriter(w, 4096*2)

	rd := iout.FnReader(func(b []byte) (int, error) { return 0, io.EOF })
	type iorwc struct {
		io.Reader
		io.Writer
		io.Closer
	}
	return io.ReadWriteCloser(&iorwc{rd, wc, wc})
}

//----------

// UI Safe
func (erow *ERow) Flash() {
	p, ok := erow.TbData.PartAtIndex(0)
	if ok {
		if len(p.Args) > 0 {
			a := p.Args[0]
			erow.Row.Toolbar.FlashIndexLen(a.Pos(), a.End()-a.Pos())
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

//----------

func (erow *ERow) SaveFileBusyCursor() {
	erow.Ed.RunAsyncBusyCursor(erow.Row, func() {
		if err := erow.Info.SaveFile(); err != nil {
			erow.Ed.Error(err)
		}
	})
}

//----------

func (erow *ERow) SyntaxComments() []*drawutil.SyntaxComment {
	ta := erow.Row.TextArea
	if d, ok := ta.Drawer.(*drawer4.Drawer); ok {
		opt := &d.Opt.SyntaxHighlight
		return opt.Comment.SCs
	}
	return nil
}

//----------
//----------
//----------

type terminalOpt struct {
	filter    bool
	keyEvents bool
}

func (t *terminalOpt) On() bool {
	return t.filter || t.keyEvents
}
