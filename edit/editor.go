package edit

import (
	"flag"
	"log"
	"os"
	"runtime/pprof"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/golang/freetype/truetype"
	"github.com/jmigpin/editor/drawutil"
	"github.com/jmigpin/editor/edit/cmdutil"
	"github.com/jmigpin/editor/edit/contentcmd"
	"github.com/jmigpin/editor/edit/toolbardata"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/xutil/wmprotocols"
	"github.com/jmigpin/editor/xutil/xgbutil"

	"github.com/howeyc/fsnotify"
)

type Editor struct {
	ui        *ui.UI
	fw        *FilesWatcher
	rowCtx    *cmdutil.RowCtx
	reopenRow *cmdutil.ReopenRow
}

func NewEditor() (*Editor, error) {
	ed := &Editor{}

	fface, err := ed.getFontFace()
	if err != nil {
		return nil, err
	}

	ui0, err := ui.NewUI(fface)
	if err != nil {
		return nil, err
	}
	ed.ui = ui0

	ed.rowCtx = cmdutil.NewRowCtx()
	ed.reopenRow = cmdutil.NewReopenRow(ed)

	// close editor when the window is deleted
	ed.ui.Win.EvReg.Add(wmprotocols.DeleteWindowEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			ed.Close()
		}})

	// setup drop support (files, dirs, ...) from other applications
	cmdutil.SetupDragNDrop(ed)

	// files watcher for visual feedback when files change
	fw, err := NewFilesWatcher()
	if err != nil {
		return nil, err
	}
	ed.fw = fw
	ed.fw.OnError = ed.Error
	ed.fw.OnEvent = ed.onFWEvent

	// set up layout toolbar
	s := "Exit | ListSessions | NewColumn | NewRow | ReopenRow | RowDirectory | FileManager | "
	ed.ui.Layout.Toolbar.SetStrClear(s, true, true)
	// execute commands on layout toolbar
	ed.ui.Layout.Toolbar.EvReg.Add(ui.TextAreaCmdEventId,
		&xgbutil.ERCallback{func(ev xgbutil.EREvent) {
			ToolbarCmdFromLayout(ed, ed.ui.Layout.Toolbar.TextArea)
		}})

	// flags
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	flag.Parse()

	// TEST cpuprofile: ~/projects/golangcode/src/github.com/jmigpin/editor/editor --cpuprofile ./p1.prof /home/jorge/documents/finances/ledger/personal.ledger

	// flags: cpuprofile
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// flags: filenames
	args := flag.Args()
	if len(args) > 0 {
		col := ed.ActiveColumn()
		for _, s := range args {
			_, err := ed.FindRowOrCreateInColFromFilepath(s, col)
			if err != nil {
				ed.Error(err)
				continue
			}
		}
	}

	//initCatchSignals(ed.onSignal)
	go ed.fw.EventLoop()
	ed.ui.EventLoop() // blocks

	return ed, nil
}
func (ed *Editor) getFontFace() (*drawutil.Face, error) {
	useGoFont := false
	fp := "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"
	font0, err := drawutil.ParseFont(fp)
	if err != nil {
		log.Println(err)
		useGoFont = true
	}
	opt := &truetype.Options{
		Size:    12,
		Hinting: font.HintingFull,
	}
	// go font
	if useGoFont {
		font0, err = truetype.Parse(goregular.TTF)
		if err != nil {
			return nil, err
		}
		opt = &truetype.Options{
			SubPixelsX: 64, // default is 4
			SubPixelsY: 64, // default is 1
			//Size:    12,
			Size: 13,
			//DPI:     72, // 0 also means 72
			Hinting: font.HintingFull,
			//GlyphCacheEntries: 4096, // still problems with concurrent drawing?
		}
	}

	fface := drawutil.NewFace(font0, opt)
	return fface, nil
}
func (ed *Editor) Close() {
	ed.fw.Close()
	ed.ui.Close()
}
func (ed *Editor) UI() *ui.UI {
	return ed.ui
}
func (ed *Editor) RowCtx() *cmdutil.RowCtx {
	return ed.rowCtx
}
func (ed *Editor) FilesWatcherAdd(filename string) error {
	return ed.fw.Add(filename)
}
func (ed *Editor) FilesWatcherRemove(filename string) error {
	return ed.fw.Remove(filename)
}

func (ed *Editor) activeRow() (*ui.Row, bool) {
	for _, c := range ed.ui.Layout.Cols.Cols {
		for _, r := range c.Rows {
			if r.Square.Active() {
				return r, true
			}
		}
	}
	return nil, false
}
func (ed *Editor) ActiveColumn() *ui.Column {
	row, ok := ed.activeRow()
	if ok {
		return row.Col
	}
	return ed.ui.Layout.Cols.LastColumnOrNew()
}
func (ed *Editor) FindRow(s string) (*ui.Row, bool) {
	cols := ed.ui.Layout.Cols
	for _, c := range cols.Cols {
		for _, r := range c.Rows {
			tsd := ed.RowToolbarStringData(r)
			v := tsd.FirstPartFilepath()
			if v == s {
				return r, true
			}
		}
	}
	return nil, false
}
func (ed *Editor) NewRow(col *ui.Column) *ui.Row {
	row := col.NewRow()
	// toolbar cmds
	row.Toolbar.EvReg.Add(ui.TextAreaCmdEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			ToolbarCmdFromRow(ed, row)
		}})
	// toolbar possible filename change
	row.Toolbar.EvReg.Add(ui.TextAreaSetTextEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			ed.updateFilesWatcher()
		}})
	// textarea content cmds
	row.TextArea.EvReg.Add(ui.TextAreaCmdEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			contentcmd.Cmd(ed, row)
		}})
	// textarea error
	row.TextArea.EvReg.Add(ui.TextAreaErrorEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			err := ev0.(error)
			ed.Error(err)
		}})
	// textarea dirty
	row.TextArea.EvReg.Add(ui.TextAreaSetTextEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			// set as dirty only if it has a filename
			tsd := ed.RowToolbarStringData(row)
			_, ok := tsd.FirstPartFilename()
			if ok {
				row.Square.SetDirty(true)
			}
		}})
	// key shortcuts
	row.EvReg.Add(ui.RowKeyPressEventId,
		&xgbutil.ERCallback{ed.onRowKeyPress})
	// close
	row.EvReg.Add(ui.RowCloseEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			ed.rowCtx.Cancel(row)
			ed.updateFilesWatcher()
			// keep it on reopen
			ed.reopenRow.Add(row)
		}})
	return row
}
func (ed *Editor) onRowKeyPress(ev0 xgbutil.EREvent) {
	ev := ev0.(*ui.RowKeyPressEvent)
	fks := ev.Key.FirstKeysym()
	m := ev.Key.Mods
	switch {
	case m.IsControl() && fks == 's':
		cmdutil.SaveRowFile(ed, ev.Row)
	case m.IsControl() && fks == 'f':
		cmdutil.FindShortcut(ed, ev.Row)
	}
}
func (ed *Editor) FindRowOrCreate(name string) *ui.Row {
	row, ok := ed.FindRow(name)
	if ok {
		return row
	}
	// new row
	row = ed.NewRow(ed.ActiveColumn())
	row.Toolbar.SetStrClear(name, true, true)
	return row
}
func (ed *Editor) FindRowOrCreateInColFromFilepath(filepath string, col *ui.Column) (*ui.Row, error) {
	row, ok := ed.FindRow(filepath)
	if ok {
		return row, nil
	}
	// new row
	content, err := filepathContent(filepath)
	if err != nil {
		return nil, err
	}
	row = ed.NewRow(col)
	p2 := toolbardata.InsertHomeTilde(filepath)
	row.Toolbar.SetStrClear(p2+" | Reload", true, true)
	row.TextArea.SetStrClear(content, true, true)
	row.Square.SetDirty(false)
	row.Square.SetCold(false)
	return row, nil
}

func (ed *Editor) RowToolbarStringData(row *ui.Row) *toolbardata.StringData {
	return toolbardata.NewStringData(row.Toolbar.Str())
}
func (ed *Editor) FilepathContent(filepath string) (string, error) {
	return filepathContent(filepath)
}

func (ed *Editor) Error(err error) {
	row := ed.FindRowOrCreate("+Errors")
	ta := row.TextArea
	// append
	a := ta.Str() + err.Error() + "\n"
	ta.SetStrClear(a, false, true)
}
func (ed *Editor) updateFilesWatcher() {
	var u []string
	for _, c := range ed.ui.Layout.Cols.Cols {
		for _, r := range c.Rows {
			tsd := ed.RowToolbarStringData(r)
			filename, ok := tsd.FirstPartFilename()
			if ok {
				u = append(u, filename)
			}
		}
	}
	ed.fw.SetFiles(u)
}

func (ed *Editor) onFWEvent(ev *fsnotify.FileEvent) {
	// on any event
	row, ok := ed.FindRow(ev.Name)
	if ok {
		row.Square.SetCold(true)
		// this func was called async, need to request tree paint
		ed.ui.RequestTreePaint()
	}
	// always update
	ed.updateFilesWatcher()
}

//func (ed *Editor) onSignal(sig os.Signal) {
//fmt.Printf("signal: %v\n", sig)
//}
