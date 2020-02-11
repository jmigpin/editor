package core

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jmigpin/editor/core/fswatcher"
	"github.com/jmigpin/editor/core/lsproto"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/drawutil"
	"github.com/jmigpin/editor/util/drawutil/drawer4"
	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
	"golang.org/x/image/font"
)

type Editor struct {
	UI                *ui.UI
	HomeVars          *HomeVars
	Watcher           fswatcher.Watcher
	RowReopener       *RowReopener
	GoDebug           *GoDebugInstance
	LSProtoMan        *lsproto.Manager
	InlineComplete    *InlineComplete
	Plugins           *Plugins
	EEvents           *EEvents // editor events (used by plugins)
	FsCaseInsensitive bool     // filesystem

	dndh *DndHandler
	ifbw *InfoFloatBoxWrap

	erowInfos map[string]*ERowInfo // use ed.ERowInfo*() to access
}

func NewEditor(opt *Options) (*Editor, error) {
	ed := &Editor{}
	ed.erowInfos = map[string]*ERowInfo{}
	ed.ifbw = NewInfoFloatBox(ed)

	// TODO: osx can have a case insensitive filesystem
	ed.FsCaseInsensitive = runtime.GOOS == "windows"

	ed.HomeVars = NewHomeVars()
	ed.RowReopener = NewRowReopener(ed)
	ed.dndh = NewDndHandler(ed)
	ed.GoDebug = NewGoDebugInstance(ed)
	ed.InlineComplete = NewInlineComplete(ed)
	ed.EEvents = NewEEvents()

	if err := ed.init(opt); err != nil {
		return nil, err
	}

	ed.initLSProto(opt)

	go ed.fswatcherEventLoop()
	ed.uiEventLoop() // blocks

	return ed, nil
}

//----------

func (ed *Editor) init(opt *Options) error {
	// fs watcher + gwatcher
	w, err := fswatcher.NewFsnWatcher()
	if err != nil {
		return err
	}
	ed.Watcher = fswatcher.NewGWatcher(w)

	ed.setupTheme(opt)
	event.UseMultiKey = opt.UseMultiKey

	// user interface
	ui0, err := ui.NewUI("Editor")
	if err != nil {
		return err
	}
	ed.UI = ui0

	// other setups
	ed.setupRootToolbar()
	ed.setupRootMenuToolbar()

	// TODO: ensure it has the window measure
	ed.EnsureOneColumn()

	// setup plugins
	setupInitialRows := true
	err = ed.setupPlugins(opt)
	if err != nil {
		ed.Error(err)
		setupInitialRows = false
	}

	if setupInitialRows {
		// enqueue setup initial rows to run after UI has window measure
		ed.UI.RunOnUIGoRoutine(func() {
			ed.setupInitialRows(opt)
		})
	}

	return nil
}

func (ed *Editor) initLSProto(opt *Options) {
	// language server protocol manager
	ed.LSProtoMan = lsproto.NewManager(ed.Message)
	for _, reg := range opt.LSProtos.regs {
		ed.LSProtoMan.Register(reg)
	}

	// auto setup gopls if there is no handler for ".go" files
	_, err := ed.LSProtoMan.LangManager("a.go")
	if err != nil { // no registration exists
		s := "go,.go,stdio,\"gopls serve\""
		reg, err := lsproto.NewRegistration(s)
		if err != nil {
			panic(err)
		}
		ed.LSProtoMan.Register(reg)
	}
}

//----------

func (ed *Editor) Close() {
	ed.LSProtoMan.Close()
	ed.UI.AppendEvent(&editorClose{})
}

//----------

func (ed *Editor) uiEventLoop() {
	defer ed.UI.Close()

	for {
		ev := ed.UI.NextEvent()
		switch t := ev.(type) {
		case error:
			log.Println(t) // in case there is no window yet
			ed.Error(t)
		case *editorClose:
			return
		case *event.WindowClose:
			return
		case *event.DndPosition:
			ed.dndh.OnPosition(t)
		case *event.DndDrop:
			ed.dndh.OnDrop(t)
		default:
			if !ed.handleGlobalShortcuts(ev) {
				if !ed.UI.HandleEvent(ev) {
					log.Printf("uievloop: unhandled event: %#v", ev)
				}
			}
		}
		ed.UI.LayoutMarkedAndSchedulePaint()
	}
}

//----------

func (ed *Editor) fswatcherEventLoop() {
	for {
		select {
		case ev, ok := <-ed.Watcher.Events():
			if !ok {
				ed.Close()
				return
			}
			switch evt := ev.(type) {
			case error:
				ed.Error(evt)
			case *fswatcher.Event:
				ed.handleWatcherEvent(evt)
			}
		}
	}
}

func (ed *Editor) handleWatcherEvent(ev *fswatcher.Event) {
	info, ok := ed.ERowInfo(ev.Name)
	if ok {
		ed.UI.RunOnUIGoRoutine(func() {
			info.UpdateDiskEvent()
		})
	}
}

//----------

func (ed *Editor) Errorf(f string, a ...interface{}) {
	ed.Error(fmt.Errorf(f, a...))
}
func (ed *Editor) Error(err error) {
	ed.Messagef("error: %v", err)
}

func (ed *Editor) Messagef(f string, a ...interface{}) {
	ed.Message(fmt.Sprintf(f, a...))
}

func (ed *Editor) Message(s string) {
	// ensure newline
	if !strings.HasSuffix(s, "\n") {
		s = s + "\n"
	}

	ed.UI.RunOnUIGoRoutine(func() {
		erow := ed.messagesERow()

		// index to make visible, get before append
		ta := erow.Row.TextArea
		index := ta.Len()

		erow.TextAreaAppendBytes([]byte(s))

		erow.MakeRangeVisibleAndFlash(index, len(s))
	})
}

//----------

func (ed *Editor) messagesERow() *ERow {
	erow, isNew := ed.ExistingOrNewERow("+Messages")
	if isNew {
		erow.ToolbarSetStrAfterNameClearHistory(" | Clear")
	}
	return erow
}

//----------

// Used for: +messages, +sessions.
func (ed *Editor) ExistingOrNewERow(name string) (_ *ERow, isnew bool) {
	info := ed.ReadERowInfo(name)
	if len(info.ERows) > 0 {
		return info.ERows[0], false
	}
	rowPos := ed.GoodRowPos()
	return NewERow(ed, info, rowPos), true
}

//----------

func (ed *Editor) ReadERowInfo(name string) *ERowInfo {
	return ReadERowInfo(ed, name)
}

func (ed *Editor) ERowInfo(name string) (*ERowInfo, bool) {
	k := ed.ERowInfoKey(name)
	info, ok := ed.erowInfos[k]
	return info, ok
}

func (ed *Editor) ERowInfos() []*ERowInfo {
	u := make([]*ERowInfo, 0, len(ed.erowInfos))
	for _, v := range ed.erowInfos { // TODO: not stable
		u = append(u, v)
	}
	return u
}

func (ed *Editor) ERowInfoKey(name string) string {
	if ed.FsCaseInsensitive {
		return strings.ToLower(name)
	}
	return name
}

func (ed *Editor) SetERowInfo(name string, info *ERowInfo) {
	k := ed.ERowInfoKey(name)
	ed.erowInfos[k] = info
}

func (ed *Editor) DeleteERowInfo(name string) {
	k := ed.ERowInfoKey(name)
	delete(ed.erowInfos, k)
}

//----------

func (ed *Editor) ERows() []*ERow {
	w := []*ERow{}
	for _, info := range ed.ERowInfos() {
		for _, e := range info.ERows {
			w = append(w, e)
		}
	}
	return w
}

//----------

func (ed *Editor) GoodRowPos() *ui.RowPos {
	return ed.UI.GoodRowPos()
}

//----------

func (ed *Editor) ActiveERow() (*ERow, bool) {
	for _, e := range ed.ERows() {
		if e.Row.HasState(ui.RowStateActive) {
			return e, true
		}
	}
	return nil, false
}

//----------

func (ed *Editor) setupRootToolbar() {
	tb := ed.UI.Root.Toolbar
	// cmd event
	tb.EvReg.Add(ui.TextAreaCmdEventId, func(ev interface{}) {
		InternalCmdFromRootTb(ed, tb)
	})
	// set str
	tb.EvReg.Add(ui.TextAreaSetStrEventId, func(ev0 interface{}) {
		ed.updateERowsToolbarsHomeVars()
	})

	s := "Exit | ListSessions | NewColumn | NewRow | Reload | "
	tb.SetStrClearHistory(s)
}

//----------

func (ed *Editor) setupRootMenuToolbar() {
	s := `CopyFilePosition
ColorTheme
CtxutilCallsState
FontRunes | FontTheme 
GoDebug 
GoRename
GotoLine 
MaximizeRow
ListDir | ListDir -hidden | ListDir -sub
ListSessions | OpenSession | DeleteSession
LSProtoCloseAll
OpenFilemanager
Reload | ReloadAll | ReloadAllFiles 
ReopenRow 
RuneCodes
SaveAllFiles
Exit | Stop | Clear`
	tb := ed.UI.Root.MainMenuButton.Toolbar
	tb.SetStrClearHistory(s)
	// cmd event
	tb.EvReg.Add(ui.TextAreaCmdEventId, func(ev interface{}) {
		InternalCmdFromRootTb(ed, tb)
	})
	// set str
	tb.EvReg.Add(ui.TextAreaSetStrEventId, func(ev0 interface{}) {
		ed.updateERowsToolbarsHomeVars()
	})
}

//----------

func (ed *Editor) updateERowsToolbarsHomeVars() {
	tb1 := ed.UI.Root.Toolbar.Str()
	tb2 := ed.UI.Root.MainMenuButton.Toolbar.Str()
	ed.HomeVars.ParseToolbarVars([]string{tb1, tb2}, ed.FsCaseInsensitive)
	for _, erow := range ed.ERows() {
		erow.updateToolbarPart0()
	}
}

//----------

func (ed *Editor) setupInitialRows(opt *Options) {
	if opt.SessionName != "" {
		OpenSessionFromString(ed, opt.SessionName)
		return
	}

	// cmd line filenames to open
	if len(opt.Filenames) > 0 {
		col := ed.UI.Root.Cols.FirstChildColumn()
		for _, filename := range opt.Filenames {
			// try to use absolute path
			u, err := filepath.Abs(filename)
			if err == nil {
				filename = u
			}

			info := ed.ReadERowInfo(filename)
			if len(info.ERows) == 0 {
				rowPos := ui.NewRowPos(col, nil)
				_, err := info.NewERow(rowPos)
				if err != nil {
					ed.Error(err)
				}
			}
		}
		return
	}

	// open current directory
	dir, err := os.Getwd()
	if err == nil {
		// create a second column (one should exist already)
		_ = ed.NewColumn()

		// open directory
		info := ed.ReadERowInfo(dir)
		cols := ed.UI.Root.Cols
		rowPos := ui.NewRowPos(cols.LastChildColumn(), nil)
		_, err := info.NewERowCreateOnErr(rowPos)
		if err != nil {
			ed.Error(err)
		}
	}
}

//----------

func (ed *Editor) setupTheme(opt *Options) {
	drawer4.WrapLineRune = rune(opt.WrapLineRune)
	drawutil.TabWidth = opt.TabWidth
	ui.ScrollBarLeft = opt.ScrollBarLeft
	ui.ScrollBarWidth = opt.ScrollBarWidth
	ui.ShadowsOn = opt.Shadows

	// color theme
	if _, ok := ui.ColorThemeCycler.GetIndex(opt.ColorTheme); !ok {
		fmt.Fprintf(os.Stderr, "unknown color theme: %v\n", opt.ColorTheme)
		os.Exit(2)
	}
	ui.ColorThemeCycler.CurName = opt.ColorTheme

	// color comments
	if opt.CommentsColor != 0 {
		ui.TextAreaCommentsColor = imageutil.IntRGBA(opt.CommentsColor)
	}

	// color strings
	if opt.StringsColor != 0 {
		ui.TextAreaStringsColor = imageutil.IntRGBA(opt.StringsColor)
	}

	// font options
	ui.TTFontOptions.Size = opt.FontSize
	ui.TTFontOptions.DPI = opt.DPI
	switch opt.FontHinting {
	case "none":
		ui.TTFontOptions.Hinting = font.HintingNone
	case "vertical":
		ui.TTFontOptions.Hinting = font.HintingVertical
	case "full":
		ui.TTFontOptions.Hinting = font.HintingFull
	default:
		fmt.Fprintf(os.Stderr, "unknown font hinting: %v\n", opt.FontHinting)
		os.Exit(2)
	}

	// font theme
	if _, ok := ui.FontThemeCycler.GetIndex(opt.Font); ok {
		ui.FontThemeCycler.CurName = opt.Font
	} else {
		// font filename
		err := ui.AddUserFont(opt.Font)
		if err != nil {
			// can't send error to UI since it's not created yet
			log.Print(err)

			// could fail and abort, but instead continue with a known font
			ui.FontThemeCycler.CurName = "regular"
		}
	}
}

//----------

func (ed *Editor) setupPlugins(opt *Options) error {
	ed.Plugins = NewPlugins(ed)
	a := strings.Split(opt.Plugins, ",")
	for _, s := range a {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		err := ed.Plugins.AddPath(s)
		if err != nil {
			return err
		}
	}
	return nil
}

//----------

func (ed *Editor) EnsureOneColumn() {
	if ed.UI.Root.Cols.ColsLayout.Spl.ChildsLen() == 0 {
		_ = ed.NewColumn()
	}
}

func (ed *Editor) NewColumn() *ui.Column {
	col := ed.UI.Root.Cols.NewColumn()
	// close
	col.EvReg.Add(ui.ColumnCloseEventId, func(ev0 interface{}) {
		ed.EnsureOneColumn()
	})
	return col
}

//----------

func (ed *Editor) handleGlobalShortcuts(ev interface{}) (handled bool) {
	switch t := ev.(type) {
	case *event.WindowInput:
		autoCloseInfo := true

		switch t2 := t.Event.(type) {
		case *event.KeyDown:
			m := t2.Mods.ClearLocks()
			if m.Is(event.ModNone) {
				switch t2.KeySym {
				case event.KSymEscape:
					ed.GoDebug.CancelAndClear()
					ed.InlineComplete.CancelAndClear()
					ed.cancelERowsContentCmds()
					autoCloseInfo = false
					ed.cancelInfoFloatBox()
					return true
				case event.KSymF1:
					autoCloseInfo = false
					ed.toggleInfoFloatBox()
					return true
				}
			}
		}

		if autoCloseInfo {
			ed.UI.Root.ContextFloatBox.AutoClose(t.Event, t.Point)
			if !ed.ifbw.ui().Visible() {
				ed.cancelInfoFloatBox()
			}
		}
	}
	return false
}

//----------

func (ed *Editor) cancelERowsContentCmds() {
	for _, erow := range ed.ERows() {
		erow.CancelContentCmd()
	}
}

//----------

func (ed *Editor) cancelInfoFloatBox() {
	ed.ifbw.Cancel()
	cfb := ed.ifbw.ui()
	cfb.Hide()
}

func (ed *Editor) toggleInfoFloatBox() {
	ed.ifbw.Cancel() // cancel previous run

	// toggle
	cfb := ed.ifbw.ui()
	cfb.Toggle()
	if !cfb.Visible() {
		return
	}

	// showInfoFloatBox

	// find ta/erow under pointer
	ta, ok := cfb.FindTextAreaUnderPointer()
	if !ok {
		cfb.Hide()
		return
	}
	erow, ok := ed.NodeERow(ta)
	if !ok {
		cfb.Hide()
		return
	}

	// show util
	show := func(s string) {
		cfb.TextArea.ClearPos()
		cfb.SetStrClearHistory(s)
		cfb.Show()
	}
	showAsync := func(s string) {
		ed.UI.RunOnUIGoRoutine(func() {
			if cfb.Visible() {
				show(s)
			}
		})
	}

	// initial ui feedback at position
	cfb.SetRefPointToTextAreaCursor(ta)
	show("Loading...")

	ed.RunAsyncBusyCursor(cfb, func() {
		// there is no timeout to complete since the context can be canceled manually

		// context based on erow context
		ctx := ed.ifbw.NewCtx(erow.ctx)

		// plugin autocomplete
		showAsync("Loading plugin...")
		err, handled := ed.Plugins.RunAutoComplete(ctx, cfb)
		if handled {
			if err != nil {
				ed.Error(err)
			}
			return
		}

		// lsproto autocomplete
		filename := ""
		switch ta {
		case erow.Row.TextArea:
			if erow.Info.IsDir() {
				filename = ".editor_directory"
			} else {
				filename = erow.Info.Name()
			}
		case erow.Row.Toolbar.TextArea:
			filename = ".editor_toolbar"
		default:
			showAsync("")
			return
		}
		// handle filename
		lang, err := ed.LSProtoMan.LangManager(filename)
		if err != nil {
			showAsync(err.Error()) // err:"no registration for..."
			return
		}
		// ui feedback while loading
		v := fmt.Sprintf("Loading lsproto(%v)...", lang.Reg.Language)
		showAsync(v)
		// lsproto autocomplete
		s, err := ed.lsprotoManAutoComplete(ctx, ta, erow)
		if err != nil {
			ed.Error(err)
			showAsync("")
			return
		}
		showAsync(s)
	})
}

func (ed *Editor) lsprotoManAutoComplete(ctx context.Context, ta *ui.TextArea, erow *ERow) (string, error) {
	tc := erow.Row.TextArea.TextCursor
	comps, err := ed.LSProtoMan.TextDocumentCompletionDetailStrings(ctx, erow.Info.Name(), tc.RW(), tc.Index())
	if err != nil {
		return "", err
	}
	s := "0 results"
	if len(comps) > 0 {
		s = strings.Join(comps, "\n")
	}
	return s, nil
}

//----------

func (ed *Editor) NodeERow(node widget.Node) (*ERow, bool) {
	for p := node.Embed().Parent; p != nil; p = p.Parent {
		if r, ok := p.Wrapper.(*ui.Row); ok {
			for _, erow := range ed.ERows() {
				if r == erow.Row {
					return erow, true
				}
			}
		}
	}
	return nil, false
}

//----------

func (ed *Editor) RunAsyncBusyCursor(node widget.Node, fn func()) {
	en := node.Embed()
	ed.UI.RunOnUIGoRoutine(func() {
		en.Cursor = event.WaitCursor
		ed.UI.QueueEmptyWindowInputEvent() // updates cursor tree
	})
	go func() {
		fn()
		ed.UI.RunOnUIGoRoutine(func() {
			en.Cursor = event.NoneCursor
			ed.UI.QueueEmptyWindowInputEvent() // updates cursor tree
		})
	}()
}

//----------

func (ed *Editor) SetAnnotations(req EdAnnotationsRequester, ta *ui.TextArea, on bool, selIndex int, entries []*drawer4.Annotation) {
	if !ed.CanModifyAnnotations(req, ta, "") {
		return
	}

	if d, ok := ta.Drawer.(*drawer4.Drawer); ok {
		d.Opt.Annotations.On = on
		d.Opt.Annotations.Selected.EntryIndex = selIndex
		d.Opt.Annotations.Entries = entries
		ta.MarkNeedsLayoutAndPaint()
	}

	// restore godebug annotations
	if req == EdAnnReqInlineComplete && !on {
		// find erow info from textarea
		for _, erow := range ed.ERows() {
			if erow.Row.TextArea == ta {
				ed.GoDebug.UpdateUIERowInfo(erow.Info)
			}
		}
	}
}

func (ed *Editor) CanModifyAnnotations(req EdAnnotationsRequester, ta *ui.TextArea, option string) bool {
	switch req {
	case EdAnnReqGoDebug:
		if option == "starting_session" {
			ed.InlineComplete.CancelAndClear()
			return true
		}
		if ed.InlineComplete.IsOn(ta) {
			return false
		}
		return true
	case EdAnnReqInlineComplete:
		return true
	default:
		panic(req)
	}
}

type EdAnnotationsRequester int

const (
	EdAnnReqGoDebug EdAnnotationsRequester = iota
	EdAnnReqInlineComplete
)

//----------

type InfoFloatBoxWrap struct {
	ed   *Editor
	ctx  context.Context
	canc context.CancelFunc
}

func NewInfoFloatBox(ed *Editor) *InfoFloatBoxWrap {
	return &InfoFloatBoxWrap{ed: ed}
}
func (ifbw *InfoFloatBoxWrap) NewCtx(ctx context.Context) context.Context {
	ifbw.Cancel() // cancel previous
	ifbw.ctx, ifbw.canc = context.WithCancel(ctx)
	return ifbw.ctx
}
func (ifbw *InfoFloatBoxWrap) Cancel() {
	if ifbw.canc != nil {
		ifbw.canc()
		ifbw.canc = nil
	}
}
func (ifbw *InfoFloatBoxWrap) ui() *ui.ContextFloatBox {
	return ifbw.ed.UI.Root.ContextFloatBox
}

//----------

type Options struct {
	Font        string
	FontSize    float64
	FontHinting string
	DPI         float64

	TabWidth     int
	WrapLineRune int

	ColorTheme     string
	CommentsColor  int
	StringsColor   int
	ScrollBarWidth int
	ScrollBarLeft  bool
	Shadows        bool

	SessionName string
	Filenames   []string

	UseMultiKey bool

	Plugins string

	LSProtos RegistrationsOpt
}

//----------

// implements flag.Value interface
type RegistrationsOpt struct {
	regs []*lsproto.Registration
}

func (ro *RegistrationsOpt) Set(s string) error {
	reg, err := lsproto.NewRegistration(s)
	if err != nil {
		return err
	}
	ro.regs = append(ro.regs, reg)
	return nil
}

func (ro *RegistrationsOpt) MustSet(s string) {
	if err := ro.Set(s); err != nil {
		panic(err)
	}
}

func (ro *RegistrationsOpt) String() string {
	u := []string{}
	for _, reg := range ro.regs {
		s := lsproto.RegistrationString(reg)
		u = append(u, s)
	}
	return fmt.Sprintf("%v", strings.Join(u, "\n"))
}

//----------

type editorClose struct{}
