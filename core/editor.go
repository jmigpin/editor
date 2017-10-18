package core

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/BurntSushi/xgb"
	"github.com/golang/freetype/truetype"
	"github.com/jmigpin/editor/core/cmdutil"
	"github.com/jmigpin/editor/core/fileswatcher"
	"github.com/jmigpin/editor/core/toolbardata"
	"github.com/jmigpin/editor/drawutil2"
	"github.com/jmigpin/editor/drawutil2/loopers"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/xgbutil/evreg"
	"github.com/jmigpin/editor/xgbutil/wmprotocols"
)

type Editor struct {
	ui    *ui.UI
	erows map[*ui.Row]*ERow
	close chan struct{}

	homeVars  toolbardata.HomeVars
	reopenRow *cmdutil.ReopenRow

	fwatcher *fileswatcher.TargetWatcher
	watch    map[string]int
}

func NewEditor(opt *Options) (*Editor, error) {
	ed := &Editor{
		erows: make(map[*ui.Row]*ERow),
		close: make(chan struct{}),
		watch: make(map[string]int),
	}

	loopers.WrapLineRune = rune(opt.WrapLineRune)
	drawutil2.TabWidth = opt.TabWidth
	ui.ScrollbarLeft = opt.ScrollbarLeft

	ui.SetScrollbarAndSquareWidth(opt.ScrollbarWidth)

	if opt.AcmeColors {
		ui.AcmeColors()
	}

	ed.reopenRow = cmdutil.NewReopenRow(ed)

	fface, err := ed.getFontFace(opt)
	if err != nil {
		return nil, err
	}
	defer fface.Close()

	ui0, err := ui.NewUI(fface)
	if err != nil {
		return nil, err
	}
	ed.ui = ui0

	// close editor when the window is deleted
	ed.ui.EvReg.Add(wmprotocols.DeleteWindowEventId, func(ev0 interface{}) {
		ed.Close()
	})

	// setup drop support (files, dirs, ...) from other applications
	cmdutil.SetupDragNDrop(ed)

	// set up layout toolbar
	s := "Exit | ListSessions | NewColumn | NewRow | ReopenRow | RowDirectory | Reload | DuplicateRow | "
	ed.ui.Layout.Toolbar.SetStrClear(s, true, true)
	// execute commands on layout toolbar
	ed.ui.Layout.Toolbar.EvReg.Add(ui.TextAreaCmdEventId, func(ev interface{}) {
		ToolbarCmdFromLayout(ed, ed.ui.Layout)
	})

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

func (ed *Editor) openInitialRows(opt *Options) {
	if opt.SessionName != "" {
		cmdutil.OpenSessionFromString(ed, opt.SessionName)
		return
	}

	// cmd line filenames to open
	if len(opt.Filenames) > 0 {
		col, _ := ed.ui.Layout.Cols.FirstChildColumn()
		for _, s := range opt.Filenames {
			_, ok := ed.FindERow(s)
			if !ok {
				erow := ed.NewERowBeforeRow(s, col, nil) // position at end
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

func (ed *Editor) getFontFace(opt *Options) (font.Face, error) {
	// test font
	// "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"

	ttf := goregular.TTF // default font

	if opt.FontFilename != "" {
		ttf2, err := ioutil.ReadFile(opt.FontFilename)
		if err != nil {
			// show error and continue with default
			log.Println(err)
		} else {
			ttf = ttf2
		}
	}

	f, err := truetype.Parse(ttf)
	if err != nil {
		return nil, err
	}

	ttOpt := &truetype.Options{
		Hinting: font.HintingFull,
		Size:    opt.FontSize,
		DPI:     opt.DPI,
	}
	fface := drawutil2.NewFace(f, ttOpt)
	return fface, nil
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

func (ed *Editor) ERows() []cmdutil.ERower {
	u := make([]cmdutil.ERower, len(ed.erows))
	i := 0
	for _, erow := range ed.erows {
		u[i] = erow
		i++
	}
	return u
}

func (ed *Editor) NewERowBeforeRow(tbStr string, col *ui.Column, nextRow *ui.Row) cmdutil.ERower {
	row := col.NewRowBefore(nextRow)
	erow := NewERow(ed, row, tbStr)

	// add/remove to erows
	ed.erows[row] = erow
	row.EvReg.Add(ui.RowCloseEventId, func(ev0 interface{}) {
		delete(ed.erows, row)

		// clears square visual queue of the duplicate that stays, if any
		erow.UpdateDuplicates()
	})

	return erow
}

func (ed *Editor) FindERow(str string) (cmdutil.ERower, bool) {
	// If iterate over ed.erows, then finderow will not be deterministic
	// Important when clicking a file name with duplicate rows present,
	// and not going to the same row consistently.

	for _, col := range ed.ui.Layout.Cols.Columns() {
		for _, row := range col.Rows() {
			erow := ed.erows[row]
			// name covers special rows, filename covers abs path
			if str == erow.Name() || str == erow.Filename() {
				return erow, true
			}
		}
	}
	return nil, false
}

func (ed *Editor) Errorf(f string, a ...interface{}) {
	ed.Error(fmt.Errorf(f, a...))
}
func (ed *Editor) Error(err error) {
	ed.Messagef("error: " + err.Error())
}

func (ed *Editor) Messagef(f string, a ...interface{}) {
	erow := ed.messagesERow()
	erow.TextAreaAppendAsync(fmt.Sprintf(f, a...) + "\n")
	//erow.Row().WarpPointer()
}
func (ed *Editor) messagesERow() cmdutil.ERower {
	s := "+Messages" // special name format
	erow, ok := ed.FindERow(s)
	if !ok {
		col, nextRow := ed.GoodColumnRowPlace()
		erow = ed.NewERowBeforeRow(s, col, nextRow)
	}
	return erow
}

func (ed *Editor) IsSpecialName(s string) bool {
	return len(s) > 0 && s[0] == '+'
}

// Used to run layout toolbar commands.
func (ed *Editor) ActiveERow() (cmdutil.ERower, bool) {
	for _, erow := range ed.erows {
		if erow.row.Square.Value(ui.SquareActive) {
			return erow, true
		}
	}
	return nil, false
}

func (ed *Editor) GoodColumnRowPlace() (*ui.Column, *ui.Row) {
	return ed.ui.Layout.GoodColumnRowPlace()
}

func (ed *Editor) eventLoop() {
	defer ed.ui.Close()

	var lastPaint time.Time
	paintIfNeeded := func() {
		painted := ed.ui.PaintIfNeeded()
		if painted {
			lastPaint = time.Now()
		}
	}

	for {
		select {
		case <-ed.close:
			goto forEnd

		case ev, _ := <-ed.ui.Events2:

			// TODO: replace this with evreg.onevent callback?

			// commented: ed.close is used
			//if !ok {
			//	goto forEnd
			//}

			ev2 := ev.(*evreg.EventWrap) // always this type for now

			switch ev2.EventId {
			case evreg.NoOpEventId:
				// do nothing, allows to check if paint is needed
			case evreg.ErrorEventId:
				err := ev2.Event.(error)
				if err, ok := err.(xgb.Error); ok {
					log.Print(err)
				} else {
					ed.Error(err)
				}
			default:
				n := ed.ui.EvReg.RunCallbacks(ev2.EventId, ev2.Event)
				if n == 0 {
					// unhandled enqueued events (mostly coming from xgb)
					ed.Errorf("%#v", ev2)
				}
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

		if len(ed.ui.Events2) == 0 {
			paintIfNeeded()
		} else {
			// ensure a paint at x frames per second
			d := time.Now().Sub(lastPaint)
			if d > time.Second/35 {
				paintIfNeeded()
			}
		}
	}
forEnd:
}

func (ed *Editor) handleWatcherEvent(ev *fileswatcher.Event) {
	//log.Printf("watcher event: %+v", ev)
	for _, erow := range ed.erows {
		if erow.Filename() == ev.Name {
			erow.UpdateState()
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
	FontFilename   string
	FontSize       float64
	DPI            float64
	ScrollbarWidth int
	AcmeColors     bool
	WrapLineRune   int
	TabWidth       int
	ScrollbarLeft  bool
	SessionName    string
	Filenames      []string
}
