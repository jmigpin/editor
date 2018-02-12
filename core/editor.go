package core

import (
	"fmt"
	"image/color"
	"log"
	"os"
	"strings"

	"github.com/jmigpin/editor/core/cmdutil"
	"github.com/jmigpin/editor/core/fileswatcher"
	"github.com/jmigpin/editor/core/toolbardata"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/drawutil"
	"github.com/jmigpin/editor/util/drawutil/loopers"
	"github.com/jmigpin/editor/util/uiutil/event"
	"golang.org/x/image/font"
)

type Editor struct {
	ui     *ui.UI
	events chan interface{}
	close  chan struct{}
	erows  map[*ui.Row]*ERow
	dndh   *cmdutil.DndHandler

	homeVars  toolbardata.HomeVars
	reopenRow *cmdutil.ReopenRow

	fwatcher *fileswatcher.TargetWatcher
	watch    map[string]int
}

func NewEditor(opt *Options) (*Editor, error) {
	ed := &Editor{
		events: make(chan interface{}, 32),
		erows:  make(map[*ui.Row]*ERow),
		close:  make(chan struct{}),
		watch:  make(map[string]int),
	}

	loopers.WrapLineRune = rune(opt.WrapLineRune)
	drawutil.TabWidth = opt.TabWidth
	ui.ScrollBarLeft = opt.ScrollBarLeft
	ui.ScrollBarWidth = opt.ScrollBarWidth

	ui.ShadowsOn = opt.Shadows

	ed.reopenRow = cmdutil.NewReopenRow(ed)

	ed.setupTheme(opt)

	var err error
	ed.ui, err = ui.NewUI(ed.events, "Editor")
	if err != nil {
		return nil, err
	}
	ed.ui.OnError = ed.Error

	// drag and drop
	ed.dndh = cmdutil.NewDndHandler(ed)

	ed.setupLayoutToolbar()

	ed.setupMenuToolbar()

	// layout home vars
	ed.homeVars.Append("~", os.Getenv("HOME"))
	cmdutil.SetupLayoutHomeVars(ed)

	// files watcher for visual feedback when files change
	ed.fwatcher, err = fileswatcher.NewTargetWatcher(nil)
	//ed.fwatcher, err = fileswatcher.NewTargetWatcher(log.Printf)
	if err != nil {
		return nil, err
	}

	ed.openInitialRows(opt)

	ed.eventLoop() // blocks

	return ed, nil
}

func (ed *Editor) setupLayoutToolbar() {
	s := "Exit | ListSessions | NewColumn | NewRow | Reload | DuplicateRow | "
	tb := ed.ui.Root.Toolbar
	tb.SetStrClear(s, true, true)
	// execute commands on layout toolbar
	tb.EvReg.Add(ui.TextAreaCmdEventId, func(ev interface{}) {
		ToolbarCmdFromLayout(ed, tb.TextArea)
	})
}

func (ed *Editor) setupMenuToolbar() {
	s := `XdgOpenDir
GotoLine | CopyFilePosition
ReopenRow | MaximizeRow
CloseColumn | CloseRow
ListDir | ListDirHidden | ListDirSub 
Reload | ReloadAll | ReloadAllFiles | SaveAllFiles
FontRunes | FontTheme | ColorTheme
ListSessions
GoDebug
Exit | Stop`
	tb := ed.ui.Root.MainMenuButton.FloatMenu.Toolbar
	tb.SetStrClear(s, true, true)
	tb.EvReg.Add(ui.TextAreaCmdEventId, func(ev interface{}) {
		ToolbarCmdFromLayout(ed, tb.TextArea)
	})

}

func (ed *Editor) setupTheme(opt *Options) {
	// color theme
	if _, ok := ui.ColorThemeCycler.GetIndex(opt.ColorTheme); !ok {
		fmt.Fprintf(os.Stderr, "unknown color theme: %v\n", opt.ColorTheme)
		os.Exit(2)
	}
	ui.ColorThemeCycler.Set(opt.ColorTheme)

	// color comments
	if opt.CommentsColor == 0 {
		ui.TextAreaCommentsColor = nil
	} else {
		v := opt.CommentsColor & 0xffffff
		r := uint8((v << 0) >> 16)
		g := uint8((v << 8) >> 16)
		b := uint8((v << 16) >> 16)
		ui.TextAreaCommentsColor = color.RGBA{r, g, b, 255}
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
		ui.FontThemeCycler.Set(opt.Font)
	} else {
		// font filename
		err := ui.AddUserFont(opt.Font)
		if err != nil {
			// TODO: send error msg to "+messages"?
			log.Print(err)

			// could fail and abort, but instead continue with a known font
			ui.FontThemeCycler.Set("regular")
		}
	}
}

func (ed *Editor) runGlobalShortcuts(ev interface{}) {
	wi, ok := ev.(*event.WindowInput)
	if !ok {
		return
	}
	p := wi.Point
	switch t := wi.Event.(type) {
	case *event.KeyDown:
		switch {
		case t.Code == event.KCodeF1:
			cmdutil.ToggleContextFloatBox(ed, p)
		case t.Code == event.KCodeEscape:
			cmdutil.DisableContextFloatBox(ed)
			cmdutil.GoDebugArgs(ed, nil, []string{"clear"})

		case t.Code == event.KCodeF3:
			cmdutil.GoDebugArgs(ed, nil, []string{"find", "-all", "first"})
		case t.Code == event.KCodeF4:
			cmdutil.GoDebugArgs(ed, nil, []string{"find", "-all", "last"})

		case t.Code == event.KCodeF5:
			cmdutil.GoDebugArgs(ed, nil, []string{"find", "-all", "prev"})
		case t.Code == event.KCodeF6:
			cmdutil.GoDebugArgs(ed, nil, []string{"find", "-all", "next"})

		case t.Code == event.KCodeF7:
			aerow, _ := ed.ActiveERower()
			cmdutil.GoDebugArgs(ed, aerow, []string{"find", "-line", "prev"})
		case t.Code == event.KCodeF8:
			aerow, _ := ed.ActiveERower()
			cmdutil.GoDebugArgs(ed, aerow, []string{"find", "-line", "next"})

		default:
			cmdutil.UpdateContextFloatBox(ed, p)
		}
	case *event.MouseDown:
		cmdutil.UpdateContextFloatBox(ed, p)
	}
}

func (ed *Editor) openInitialRows(opt *Options) {
	if opt.SessionName != "" {
		cmdutil.OpenSessionFromString(ed, opt.SessionName)
		return
	}

	// cmd line filenames to open
	if len(opt.Filenames) > 0 {
		col := ed.ui.Root.Cols.FirstChildColumn()
		for _, s := range opt.Filenames {
			erows := ed.FindERowers(s)
			if len(erows) == 0 {
				erow := ed.NewERowerBeforeRow(s, col, nil) // position at end
				err := erow.LoadContentClear()
				if err != nil {
					ed.Error(err)
					continue
				}
			}
		}
		return
	}

	// start with 2 colums and a current directory row on 2nd column
	cols := ed.ui.Root.Cols
	_ = cols.NewColumn() // add second column
	col := cols.LastChildColumn()
	dir, err := os.Getwd()
	if err == nil {
		cmdutil.OpenDirectoryRow(ed, dir, col, nil)
	}
}

func (ed *Editor) Close() {
	ed.fwatcher.Close()
	close(ed.close)
}
func (ed *Editor) UI() *ui.UI {
	return ed.ui
}
func (ed *Editor) HomeVars() *toolbardata.HomeVars {
	return &ed.homeVars
}

// Order is not consistent.
func (ed *Editor) ERowers() []cmdutil.ERower {
	u := make([]cmdutil.ERower, 0, len(ed.erows))
	for _, erow := range ed.erows {
		u = append(u, erow)
	}
	return u
}

func (ed *Editor) NewERowerBeforeRow(tbStr string, col *ui.Column, nextRow *ui.Row) cmdutil.ERower {
	row := col.NewRowBefore(nextRow)
	return NewERow(ed, row, tbStr)
}

func (ed *Editor) RegisterERow(e *ERow) {
	ed.erows[e.row] = e
}
func (ed *Editor) UnregisterERow(e *ERow) {
	delete(ed.erows, e.row)
}

func (ed *Editor) FindERows(str string) []*ERow {
	// find in col/row order to have consistent results order
	var a []*ERow
	for _, col := range ed.ui.Root.Cols.Columns() {
		for _, row := range col.Rows() {
			erow, ok := ed.erows[row]

			// Row is not yet in the erow mapping due to creating a new erow. Could only happen in a concurrent scenario.
			if !ok {
				continue
			}

			// name covers special rows, filename covers abs path
			if str == erow.Name() || str == erow.Filename() {
				a = append(a, erow)
			}
		}
	}
	return a
}

func (ed *Editor) FindERowers(str string) []cmdutil.ERower {
	u := ed.FindERows(str)
	a := make([]cmdutil.ERower, 0, len(u))
	for _, e := range u {
		a = append(a, e)
	}
	return a
}

func (ed *Editor) Errorf(f string, a ...interface{}) {
	ed.Error(fmt.Errorf(f, a...))
}
func (ed *Editor) Error(err error) {
	ed.Messagef("error: %v", err.Error())
}

func (ed *Editor) Messagef(f string, a ...interface{}) {
	ed.UI().RunOnUIGoRoutine(func() {
		erow := ed.messagesERow()

		// add newline
		s := fmt.Sprintf(f, a...)
		if !strings.HasSuffix(s, "\n") {
			s = s + "\n"
		}

		// index to make visible, get before append
		ta := erow.Row().TextArea
		index := len(ta.Str()) + 1 // +1 for "\n" that is inserted above

		erow.(*ERow).textAreaAppend(s)

		// auto scroll to show the new message
		ta.MakeIndexVisible(index)

		erow.Flash() // TODO: need to flash since if too small, the content flash won't show
		ta.FlashIndexLine(index)
	})
}
func (ed *Editor) messagesERow() cmdutil.ERower {
	rowName := "+Messages" // special name format
	erows := ed.FindERowers(rowName)
	if len(erows) > 0 {
		return erows[0]
	}
	col, nextRow := ed.GoodColumnRowPlace()
	return ed.NewERowerBeforeRow(rowName+" | Clear", col, nextRow)
}

func (ed *Editor) IsSpecialName(s string) bool {
	return len(s) > 0 && s[0] == '+'
}

// Used to run layout toolbar commands.
func (ed *Editor) ActiveERower() (cmdutil.ERower, bool) {
	for _, erow := range ed.erows {
		if erow.row.HasState(ui.ActiveRowState) {
			return erow, true
		}
	}
	return nil, false
}

func (ed *Editor) GoodColumnRowPlace() (*ui.Column, *ui.Row) {
	return ed.ui.Root.GoodColumnRowPlace()
}

func (ed *Editor) eventLoop() {
	defer ed.ui.Close() // TODO: review

	for {
		select {
		case <-ed.close:
			goto forEnd

		case ev := <-ed.events:
			switch t := ev.(type) {
			case error:
				ed.Error(t)
			case *event.WindowClose:
				ed.Close() // TODO: review
			case *event.DndPosition:
				ed.dndh.OnPosition(t)
			case *event.DndDrop:
				ed.dndh.OnDrop(t)
			default:
				ed.ui.HandleEvent(ev)
				ed.runGlobalShortcuts(ev)
			}

		case ev, ok := <-ed.fwatcher.Events:
			if !ok {
				break
			}
			switch ev2 := ev.(type) {
			case error:
				ed.Error(ev2)
			case *fileswatcher.Event:
				ed.handleWatcherEvent(ev2)
			}
		}

		ed.ui.PaintIfTime()
	}
forEnd:
}

func (ed *Editor) handleWatcherEvent(ev *fileswatcher.Event) {
	for _, erow := range ed.erows {
		if erow.Filename() == ev.Name {
			erow.UpdateStateAndDuplicates()
		}
	}
}

func (ed *Editor) IncreaseWatch(filename string) {
	_, ok := ed.watch[filename]
	if !ok {
		ed.fwatcher.Add(filename)
	}
	ed.watch[filename]++
}
func (ed *Editor) DecreaseWatch(filename string) {
	c, ok := ed.watch[filename]
	if !ok {
		return
	}
	c--
	if c == 0 {
		delete(ed.watch, filename)
		ed.fwatcher.Remove(filename)
	} else {
		ed.watch[filename] = c
	}
}

type Options struct {
	Font           string
	FontSize       float64
	FontHinting    string
	DPI            float64
	ScrollBarWidth int
	ScrollBarLeft  bool
	ColorTheme     string
	CommentsColor  int
	WrapLineRune   int
	TabWidth       int
	Shadows        bool
	SessionName    string
	Filenames      []string
}
