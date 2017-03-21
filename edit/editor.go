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
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/xutil/wmprotocols"
	"github.com/jmigpin/editor/xutil/xgbutil"
)

type Editor struct {
	ui        *ui.UI
	fw        *FilesWatcher
	reopenRow *cmdutil.ReopenRow
	erows     map[*ui.Row]*ERow
}

func NewEditor() (*Editor, error) {
	ed := &Editor{
		erows: make(map[*ui.Row]*ERow),
	}
	ed.reopenRow = cmdutil.NewReopenRow(ed)

	fface, err := ed.getFontFace()
	if err != nil {
		return nil, err
	}

	ui0, err := ui.NewUI(fface)
	if err != nil {
		return nil, err
	}
	ed.ui = ui0

	// close editor when the window is deleted
	ed.ui.Win.EvReg.Add(wmprotocols.DeleteWindowEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			ed.Close()
		}})

	// setup drop support (files, dirs, ...) from other applications
	cmdutil.SetupDragNDrop(ed)

	// files watcher for visual feedback when files change
	fw, err := NewFilesWatcher(ed)
	if err != nil {
		return nil, err
	}
	ed.fw = fw
	//ed.fw.OnError = ed.Error
	//ed.fw.OnEvent = ed.onFWEvent

	// set up layout toolbar
	s := "Exit | ListSessions | NewColumn | NewRow | ReopenRow | RowDirectory | FileManager | "
	ed.ui.Layout.Toolbar.SetStrClear(s, true, true)
	// execute commands on layout toolbar
	ed.ui.Layout.Toolbar.EvReg.Add(ui.TextAreaCmdEventId,
		&xgbutil.ERCallback{func(ev xgbutil.EREvent) {
			ToolbarCmdFromLayout(ed, ed.ui.Layout)
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
			erow := ed.FindERowOrCreate(s, col)
			err := erow.LoadContentClear()
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

func (ed *Editor) activeERow() (*ERow, bool) {
	for _, erow := range ed.erows {
		if erow.row.Square.Value(ui.SquareActive) {
			return erow, true
		}
	}
	return nil, false
}
func (ed *Editor) ActiveColumn() *ui.Column {
	// TODO: who calls this, needs to check if opening a dir, then it should be on same col, else, on best space

	//row, ok := ed.activeRow()
	//if ok {
	//return row.Col
	//}
	//return ed.ui.Layout.Cols.LastColumnOrNew()

	return ed.ui.Layout.Cols.ColumnWithBestSpaceForNewRow()
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
func (ed *Editor) NewERow(col *ui.Column) cmdutil.ERower {
	row := col.NewRow()
	erow := NewERow(ed, row)
	ed.erows[row] = erow
	// on row close - clear from erows
	row.EvReg.Add(ui.RowCloseEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			delete(ed.erows, row)
		}})
	// key shortcuts
	row.EvReg.Add(ui.RowKeyPressEventId,
		&xgbutil.ERCallback{ed.onRowKeyPress})
	return erow
}
func (ed *Editor) onRowKeyPress(ev0 xgbutil.EREvent) {
	ev := ev0.(*ui.RowKeyPressEvent)
	fks := ev.Key.FirstKeysym()
	m := ev.Key.Mods
	switch {
	case m.IsControl() && fks == 's':
		erow, ok := ed.erows[ev.Row]
		if !ok {
			panic("!")
		}
		cmdutil.SaveRowFile(erow)
	case m.IsControl() && fks == 'f':
		erow, ok := ed.erows[ev.Row]
		if !ok {
			panic("!")
		}
		cmdutil.FindShortcut(erow)
	}
}
func (ed *Editor) FindERow(s string) (cmdutil.ERower, bool) {
	for _, erow := range ed.erows {
		tsd := erow.ToolbarSD()
		s1 := tsd.FirstPartFilepath()
		if s1 == s {
			return erow, true
		}
	}
	return nil, false
}
func (ed *Editor) FindERowOrCreate(str string, col *ui.Column) cmdutil.ERower {
	erow, ok := ed.FindERow(str)
	if ok {
		return erow
	}
	erow = ed.NewERow(col)
	erow.Row().Toolbar.SetStrClear(str, true, true)
	return erow
}

func (ed *Editor) Error(err error) {
	col := ed.ActiveColumn()
	erow := ed.FindERowOrCreate("+Errors", col)
	erow.TextAreaAppend(err.Error() + "\n")
}

//func (ed *Editor) onSignal(sig os.Signal) {
//fmt.Printf("signal: %v\n", sig)
//}
