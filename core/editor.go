package core

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmigpin/editor/core/fswatcher"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/drawutil"
	"github.com/jmigpin/editor/util/drawutil/drawer3"
	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/uiutil/event"
	"golang.org/x/image/font"
)

type Editor struct {
	UI          *ui.UI
	HomeVars    *HomeVars
	Watcher     fswatcher.Watcher
	RowReopener *RowReopener
	ERowInfos   map[string]*ERowInfo
	Plugins     *Plugins

	dndh *DndHandler
}

func NewEditor(opt *Options) (*Editor, error) {
	ed := &Editor{
		ERowInfos: map[string]*ERowInfo{},
	}

	ed.HomeVars = NewHomeVars()
	ed.RowReopener = NewRowReopener(ed)
	ed.dndh = NewDndHandler(ed)

	if err := ed.init(opt); err != nil {
		return nil, err
	}

	GoDebugInit(ed)

	go ed.fswatcherEventLoop()
	ed.UI.EventLoop() // blocks

	return ed, nil
}

//----------

func (ed *Editor) init(opt *Options) error {
	// fs watcher + targetwatcher
	w, err := fswatcher.NewFsnWatcher()
	if err != nil {
		return err
	}
	*w.OpMask() = fswatcher.Create |
		fswatcher.Remove |
		fswatcher.Modify |
		fswatcher.Rename
	ed.Watcher = fswatcher.NewTargetWatcher(w)

	ed.setupTheme(opt)
	event.UseMultiKey = opt.UseMultiKey

	// user interface
	ui0, err := ui.NewUI("Editor")
	if err != nil {
		return err
	}
	ed.UI = ui0
	ed.UI.OnError = ed.Error
	ed.UI.OnEvent = ed.onUIEvent

	// other setups
	ed.setupRootToolbar()
	ed.setupRootMenuToolbar()
	ed.setupPlugins(opt)

	// TODO: ensure it has the window measure
	// enqueue setup initial rows to run after UI has window measure
	ed.EnsureOneColumn()
	ed.UI.RunOnUIGoRoutine(func() {
		ed.setupInitialRows(opt)
	})

	return nil
}

//----------

func (ed *Editor) Close() {
	ed.UI.Close()
}

//----------

func (ed *Editor) onUIEvent(ev interface{}) {
	switch t := ev.(type) {
	case *event.DndPosition:
		ed.dndh.OnPosition(t)
	case *event.DndDrop:
		ed.dndh.OnDrop(t)
	default:
		h := ed.handleGlobalShortcuts(ev)
		if h == event.NotHandled {
			ed.UI.HandleEvent(ev)
		}
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
	name := ev.JoinNames() // handle event target (join names)
	info, ok := ed.ERowInfos[name]
	if ok {
		// TODO: forcing to run on ui goroutine should be done at a lower level
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
	ed.Messagef("error: %v", err.Error())
}

func (ed *Editor) Messagef(f string, a ...interface{}) {
	// ensure newline
	s := fmt.Sprintf(f, a...)
	if !strings.HasSuffix(s, "\n") {
		s = s + "\n"
	}

	ed.UI.RunOnUIGoRoutine(func() {
		erow := ed.messagesERow()

		// index to make visible, get before append
		ta := erow.Row.TextArea
		index := len(ta.Str())

		erow.textAreaAppend(s)

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
	info, ok := ed.ERowInfos[name]
	if ok {
		info.readFileInfo()
		return info
	}
	return NewERowInfo(ed, name)
}

//----------

func (ed *Editor) ERows() []*ERow {
	w := []*ERow{}
	for _, info := range ed.ERowInfos {
		for _, e := range info.ERows {
			w = append(w, e)
		}
	}
	return w
}

//----------

func (ed *Editor) GoodRowPos() *ui.RowPos {
	return ed.UI.Root.GoodRowPos()
}

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
		RootToolbarCmd(ed, tb)
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
	s := `XdgOpenDir
GotoLine | CopyFilePosition
ReopenRow | MaximizeRow | ToggleRowHBar
CloseColumn | CloseRow
ListDir | ListDir -hidden | ListDir -sub
Reload | ReloadAll | ReloadAllFiles | SaveAllFiles
FontRunes | FontTheme | ColorTheme
GoDebug | GoRename
ListSessions
Exit | Stop | Clear`
	tb := ed.UI.Root.MainMenuButton.Toolbar
	tb.SetStrClearHistory(s)
	// cmd event
	tb.EvReg.Add(ui.TextAreaCmdEventId, func(ev interface{}) {
		RootToolbarCmd(ed, tb)
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
	ed.HomeVars.ParseToolbarVars(tb1, tb2)
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
	drawer3.WrapLineRune = rune(opt.WrapLineRune)
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

func (ed *Editor) setupPlugins(opt *Options) {
	ed.Plugins = NewPlugins(ed)
	a := strings.Split(opt.Plugins, ",")
	for _, s := range a {
		path := strings.TrimSpace(s)
		if len(path) == 0 {
			continue
		}
		ed.Plugins.AddPath(path)
	}
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

func (ed *Editor) handleGlobalShortcuts(ev interface{}) event.Handle {
	//fmt.Printf("global shortcut %#v\n", ev)

	switch t := ev.(type) {
	case *event.WindowInput:
		switch t2 := t.Event.(type) {
		case *event.KeyUp:
			m := t2.Mods.ClearLocks()
			if m.Is(event.ModNone) {
				if t2.KeySym == event.KSymEscape {
					GoDebugStop(ed)
					return event.Handled
				}
			}
		}
	}
	return event.NotHandled
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
	ScrollBarWidth int
	ScrollBarLeft  bool
	Shadows        bool

	SessionName string
	Filenames   []string

	UseMultiKey bool

	Plugins string
}
