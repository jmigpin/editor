package core

import (
	"fmt"
	"image"
	"log"
	"os"

	"golang.org/x/image/font"

	"github.com/jmigpin/editor/core/cmdutil"
	"github.com/jmigpin/editor/core/fileswatcher"
	"github.com/jmigpin/editor/core/toolbardata"
	"github.com/jmigpin/editor/drawutil2"
	"github.com/jmigpin/editor/drawutil2/loopers"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/uiutil/event"
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
	drawutil2.TabWidth = opt.TabWidth
	ui.ScrollbarLeft = opt.ScrollbarLeft

	ui.SetScrollbarAndSquareWidth(opt.ScrollbarWidth)

	switch opt.ColorTheme {
	case "light":
		ui.LightThemeColors()
	case "dark":
		ui.DarkThemeColors()
	case "acme":
		ui.AcmeThemeColors()
	default:
		ui.LightThemeColors()
	}

	ui.ShadowsOn = opt.Shadows

	ed.reopenRow = cmdutil.NewReopenRow(ed)

	// font
	ui.FontOpt.Hinting = font.HintingFull
	ui.FontOpt.Size = opt.FontSize
	ui.FontOpt.DPI = opt.DPI
	switch opt.Font {
	case "regular":
		ui.RegularFont()
	case "medium":
		ui.MediumFont()
	case "mono":
		ui.MonoFont()
	default:
		filename := opt.Font
		err := ui.SetNamedFont(filename)
		if err != nil {
			log.Print(err)
			ui.RegularFont()
		}
	}

	ui0, err := ui.NewUI(ed.events, "Editor")
	if err != nil {
		return nil, err
	}
	ui0.OnError = ed.Error
	ed.ui = ui0

	// drag and drop
	ed.dndh = cmdutil.NewDndHandler(ed)

	ed.setupLayoutToolbar()

	ed.setupMenuToolbar()

	ed.setupGlobalShortcuts()

	// layout home vars
	ed.homeVars.Append("~", os.Getenv("HOME"))
	cmdutil.SetupLayoutHomeVars(ed)

	// files watcher for visual feedback when files change
	w, err := fileswatcher.NewTargetWatcher(nil)
	//w, err := fileswatcher.NewTargetWatcher(log.Printf)
	if err != nil {
		return nil, err
	}
	ed.fwatcher = w

	ed.openInitialRows(opt)

	ed.eventLoop() // blocks

	return ed, nil
}

func (ed *Editor) setupLayoutToolbar() {
	s := "Exit | ListSessions | NewColumn | NewRow | Reload | DuplicateRow | "
	tb := ed.ui.Layout.Toolbar
	tb.SetStrClear(s, true, true)
	// execute commands on layout toolbar
	tb.EvReg.Add(ui.TextAreaCmdEventId, func(ev interface{}) {
		ToolbarCmdFromLayout(ed, tb.TextArea)
	})
}

func (ed *Editor) setupMenuToolbar() {
	s := `XdgOpenDir
GotoLine | CopyFilePosition
RowDirectory | ReopenRow | MaximizeRow
CloseColumn | CloseRow
ListDir | ListDirHidden | ListDirSub 
Reload | ReloadAll | ReloadAllFiles
SaveAllFiles
FontRunes | FontTheme | ColorTheme
ListSessions
Exit | Stop`
	tb := ed.ui.Layout.MainMenuButton.FloatMenu.Toolbar
	tb.SetStrClear(s, true, true)
	tb.EvReg.Add(ui.TextAreaCmdEventId, func(ev interface{}) {
		ToolbarCmdFromLayout(ed, tb.TextArea)
	})

}

func (ed *Editor) setupGlobalShortcuts() {
	ed.ui.AfterInputEvent = func(ev interface{}, p image.Point) {
		switch t := ev.(type) {
		case *event.KeyDown:
			switch {
			case t.Code == event.KCodeF1:
				cmdutil.ToggleContextFloatBox(ed, p)
			default:
				cmdutil.UpdateContextFloatBox(ed, p)
			}
		case *event.MouseDown:
			cmdutil.UpdateContextFloatBox(ed, p)
		}
	}
}

func (ed *Editor) openInitialRows(opt *Options) {
	if opt.SessionName != "" {
		cmdutil.OpenSessionFromString(ed, opt.SessionName)
		return
	}

	// cmd line filenames to open
	if len(opt.Filenames) > 0 {
		col, _ := ed.ui.Layout.Cols.FirstChildColumn()
		for _, s := range opt.Filenames {
			_, ok := ed.FindERower(s)
			if !ok {
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
	cols := ed.ui.Layout.Cols
	_ = cols.NewColumn()
	col, ok := cols.LastChildColumn()
	if ok {
		dir, err := os.Getwd()
		if err == nil {
			cmdutil.OpenDirectoryRow(ed, dir, col, nil)
		}
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

func (ed *Editor) ERowers() []cmdutil.ERower {
	u := make([]cmdutil.ERower, len(ed.erows))
	i := 0
	for _, erow := range ed.erows {
		u[i] = erow
		i++
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
	// find in col/row order to have consistent results
	var a []*ERow
	for _, col := range ed.ui.Layout.Cols.Columns() {
		for _, row := range col.Rows() {
			erow, ok := ed.erows[row]
			if !ok {
				// row is not yet in the mapping (creating a new erow)
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
	a := make([]cmdutil.ERower, len(u))
	for i, e := range u {
		a[i] = e
	}
	return a
}

// TODO: rename to FindFirstERower?
func (ed *Editor) FindERower(str string) (cmdutil.ERower, bool) {
	a := ed.FindERowers(str)
	if len(a) == 0 {
		return nil, false
	}
	return a[0], true
}

func (ed *Editor) Errorf(f string, a ...interface{}) {
	ed.Error(fmt.Errorf(f, a...))
}
func (ed *Editor) Error(err error) {
	//log.Printf("%v", err)
	ed.Messagef("error: " + err.Error())
}

func (ed *Editor) Messagef(f string, a ...interface{}) {
	erow := ed.messagesERow()
	erow.TextAreaAppendAsync(fmt.Sprintf(f, a...) + "\n")
	erow.Flash()
}
func (ed *Editor) messagesERow() cmdutil.ERower {
	s := "+Messages" // special name format
	erow, ok := ed.FindERower(s)
	if !ok {
		col, nextRow := ed.GoodColumnRowPlace()
		erow = ed.NewERowerBeforeRow(s, col, nextRow)
	}
	return erow
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
	return ed.ui.Layout.GoodColumnRowPlace()
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

		ed.ui.PaintIfNeeded()
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
	DPI            float64
	ScrollbarWidth int
	ColorTheme     string
	WrapLineRune   int
	TabWidth       int
	ScrollbarLeft  bool
	SessionName    string
	Shadows        bool
	Filenames      []string
}
