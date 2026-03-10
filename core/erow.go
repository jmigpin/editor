package core

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/jmigpin/editor/core/termemu"
	"github.com/jmigpin/editor/core/toolbarparser"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/drawutil"
	"github.com/jmigpin/editor/util/drawutil/drawer4"
	"github.com/jmigpin/editor/util/fontutil"
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
	scrollMode          string

	runOpts ERowRunOpts

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
	erow, err := NewERow(info, rowPos)
	if err != nil {
		return nil, err
	}
	return erow, erow.Load()
}

// Allows creating rows in place even if a file/dir doesn't exist anymore (ex: show non-existent files rows in a saved session).
func NewLoadedERowOrNewBasic(info *ERowInfo, rowPos *ui.RowPos) *ERow {
	erow, err := NewLoadedERow(info, rowPos)
	if err != nil {
		return NewBasicERow(info, rowPos)
	}
	return erow
}

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

// a new viable erow, not yet loaded, or an error without instantiating the erow
func NewERow(info *ERowInfo, rowPos *ui.RowPos) (*ERow, error) {
	switch {
	case info.IsSpecial():
		// there can be only one instance of a special row
		if len(info.ERows) > 0 {
			return nil, fmt.Errorf("special row already exists: %v", info.Name())
		}
		erow := NewBasicERow(info, rowPos)
		return erow, nil

	case info.IsDir():
		if err := info.checkOpen(); err != nil {
			return nil, err // can't read from fs
		}

		erow := NewBasicERow(info, rowPos)
		return erow, nil

	case info.IsFileButNotDir():
		if _, ok := info.FirstERow(); !ok { // can't read from existing row
			if err := info.checkOpen(); err != nil {
				return nil, err // can't read from fs
			}
		}

		erow := NewBasicERow(info, rowPos)
		return erow, nil

	case info.FileInfoErr() != nil:
		return nil, info.FileInfoErr()

	default:
		return nil, errors.New("unexpected erow type")
	}
}

func (erow *ERow) Load() error {
	return erow.Reload2(true)
}
func (erow *ERow) Reload() error {
	return erow.Reload2(false)
}

func (erow *ERow) Reload2(firstLoad bool) error {
	if !firstLoad && erow.Row.HasState(ui.RowStateExecuting) {
		return fmt.Errorf("currently running a cmd so can't reload: %s", erow.Info.Name())
	}

	switch {
	case erow.Info.IsSpecial():
		if erow.Info.Name() == "+Sessions" {
			ListSessions(erow.Ed)
		}
		return nil

	case erow.Info.IsDir():
		ListDirERow(erow, erow.Info.Name(), false, true)
		return nil

	case erow.Info.IsFileButNotDir():
		if firstLoad {
			// read content from existing row
			if erow0, ok := erow.Info.FirstERow(); ok {
				if erow0 != erow {
					// use with existing content
					erow.Info.setRWFromMaster(erow0)
					return nil
				}
			}
		}

		// load
		b, err := erow.Info.readFsFile()
		if err != nil {
			return err
		}

		// update data
		erow.Info.setSavedHash(erow.Info.fileData.fs.hash, len(b))

		// new erow (no other rows exist)
		if firstLoad {
			erow.Row.TextArea.SetBytesClearHistory(b)
		} else {
			erow.Info.SetRowsBytes(b)
		}
		return nil
	default:
		info := erow.Info
		err := fmt.Errorf("unable to load erow: %v", info.Name())
		if info.fiErr != nil {
			err = fmt.Errorf("%v: %w", err, info.fiErr)
		}
		return err
	}
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

	//// textarea layout for console
	//row.TextArea.EvReg.Add(ui.TextAreaLayoutEventId, func(ev0 any) {
	//	ev := ev0.(*ui.TextAreaLayoutEvent)
	//	_ = ev
	//	updateConsoleFontSize(erow)
	//})

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

			//case mods.Is(event.ModCtrl|event.ModShift) && evt.KeySym == event.KSymF:
			//	// internal cmd
			//	str := "FillAssist -template"
			//	data := toolbarparser.Parse(str)
			//	part, _ := data.PartAtIndex(0)
			//	internalOrExternalCmd(erow.Ed, part, erow)

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
	var userFont struct{ name, size bool }
	if v, ok := vmap["$font"]; ok {
		userFont.name, userFont.size, _ = erow.setVarFontTheme(v)
	}
	if clear := !userFont.name && !userFont.size; clear {
		erow.Row.TextArea.SetThemeFontFace(nil)
	}

	// $scrollMode: reset before $terminal
	erow.scrollMode = ""

	// $terminal
	// terminal font face: unless the user defined a font, run with a monospace font
	fface0 := erow.Row.TextArea.TreeThemeFontFace()
	fface1 := fface0
	if !userFont.name && !fface1.TestIsMono() {
		fface1 = fontutil.DefaultMonoFont().FontFace(fface1.Opts)
	}
	erow.runOpts = ERowRunOpts{ // reset
		fface:        fface1,
		ffaceRestore: fface0,
		useGrayscale: true,
	}
	if erow.Info.IsDir() {
		if v, ok := vmap["$terminal"]; ok {
			u := strings.Split(v, ",")
			for _, k := range u {
				if err := erow.applyTerminalOpt(k); err != nil {
					// TODO: can't error like this since it will be outputing errors while typing an option (the parse is done on every input)
					//erow.Ed.Error(err)
				}
			}
			//updateConsoleFontSize(erow, nil)
		}
	}

	// $scrollMode: auto/top/""
	if v, ok := vmap["$scrollMode"]; ok {
		erow.scrollMode = v
	}
}

func (erow *ERow) setVarFontTheme(s string) (bool, bool, error) {
	w := strings.SplitN(s, ",", 2)

	// font name arg
	name := (*string)(nil)
	if s := strings.TrimSpace(w[0]); s != "" {
		name = &s
	}

	// font size arg
	size := (*float64)(nil)
	if len(w) > 1 {
		v, err := strconv.ParseFloat(w[1], 64)
		if err != nil {
			// commented: ignore error
			//return false,false,err
		} else {
			// start accepting font size only at 2, allows typing 1x without affecting the rendering
			if v >= 2 {
				size = &v
			}
		}
	}

	//----------

	ta := erow.Row.TextArea

	// use parent node face options (also inherits dpi)
	face := ta.Parent.TreeThemeFontFace()
	fopts2 := face.Opts // copy

	if size != nil {
		fopts2.SetSize(*size)
	}

	ff := (*fontutil.FontFace)(nil)
	validName := false
	if name != nil {
		ff2, err := ui.ThemeFontFace2(*name, fopts2)
		if err == nil {
			ff = ff2
			validName = true
		}
	}

	if ff == nil {
		if size != nil { // change only size
			ff = face.Font.FontFace(fopts2)
		} else {
			return false, false, errors.New("unable to load name and missing size")
		}
	}

	ta.SetThemeFontFace(ff)

	return validName, size != nil, nil
}

func (erow *ERow) applyTerminalOpt(opt string) error {
	topt := &erow.runOpts

	opt = strings.ToLower(strings.TrimSpace(opt))

	set := true
	if strings.HasPrefix(opt, "no-") { // support negation
		set = false
		opt = opt[3:]
	}

	// aliases - old options
	alias := map[string]string{
		"f": "raw", // old "filter" option
		"k": "kb",
	}
	if a, ok := alias[opt]; ok {
		opt = a
	}

	switch {
	case opt == "debug":
		topt.emuOpts.Debug = true

	case opt == "grayscale":
		topt.useGrayscale = set
	case opt == "color":
		topt.useGrayscale = !set

	case opt == "pty":
		topt.pty = set
	case opt == "kb":
		topt.forwardKb = set
	case opt == "mouse":
		topt.forwardMouse = set

	case strings.HasPrefix(opt, "rows="):
		if set {
			if v, err := strconv.Atoi(opt[5:]); err == nil {
				topt.fixedRows = v
			}
		} else {
			topt.fixedRows = 0
		}
	case opt == "rows": // ignore "rows" without value, but support "no-rows"
		if !set {
			topt.fixedRows = 0
		}

	case opt == "raw":
		return topt.emuOpts.Mode.SetBool(set, termemu.ModeRaw)
	case opt == "plain":
		return topt.emuOpts.Mode.SetBool(set, termemu.ModePlain)
	case opt == "grid":
		return topt.emuOpts.Mode.SetBool(set, termemu.ModeGrid)

	case opt == "emu": // pre-set options
		if err := topt.emuOpts.Mode.SetBool(set, termemu.ModeGrid); err != nil {
			return err
		}
		if set {
			topt.pty = true
			topt.forwardKb = true
			topt.forwardMouse = true
			//erow.scrollMode = "auto" // annoying at times
		}
		return nil

	default:
		return fmt.Errorf("unknown $terminal option: %q\n\t%s", opt, erow.Info.Name())
	}

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
	return erow.OverwriteBytesClearHistory(ta.RW().Max(), 0, p)
}
func (erow *ERow) OverwriteBytesClearHistory(i, del int, p []byte) error {
	ta := erow.Row.TextArea

	scrollDown := false
	if erow.scrollMode == "auto" {
		if ta.IndexVisible(ta.RW().Max()) {
			scrollDown = true
		}
	}

	if err := ta.OverwriteBytesClearHistory(i, del, p); err != nil {
		return err
	}

	switch {
	case scrollDown:
		ta.MakeRangeVisible2(ta.RW().Max(), 0,
			//drawutil.RAlignBottom)
			drawutil.RAlignKeepOrBottom)
	case erow.scrollMode == "top":
		ta.MakeRangeVisible2(0, 0, drawutil.RAlignTop)
	}
	return nil
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

type ERowRunOpts struct {
	pty          bool // run under a pseudo-terminal
	forwardKb    bool // forward keyboard events to the process
	forwardMouse bool // forward mouse events to the process

	fixedRows int // if > 0, use fixed terminal height

	emuOpts termemu.Opts

	useGrayscale bool

	fface        *fontutil.FontFace
	ffaceRestore *fontutil.FontFace
}
