package edit

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/golang/freetype/truetype"
	"github.com/jmigpin/editor/drawutil"
	"github.com/jmigpin/editor/edit/cmdutil"
	"github.com/jmigpin/editor/edit/toolbardata"
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
	// possible x errors
	ed.ui.Win.EvReg.Add(xgbutil.XErrorEventId,
		&xgbutil.ERCallback{func(ev xgbutil.EREvent) {
			ed.Errorf("xerror: %v", ev)
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
	// home vars
	ed.ui.Layout.Toolbar.EvReg.Add(ui.TextAreaSetStrEventId,
		&xgbutil.ERCallback{ed.onLayoutToolbarSetStr})

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
		col := ed.ui.Layout.Cols.Cols[0]
		for _, s := range args {
			_, ok := ed.FindERow(s)
			if !ok {
				rowIndex := len(col.Rows)
				erow := ed.NewERow(s, col, rowIndex)
				err := erow.LoadContentClear()
				if err != nil {
					// TODO: can't show errors yet?
					//ed.Error(err)
					continue
				}
			}
		}
	}

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

func (ed *Editor) ERows() []cmdutil.ERower {
	u := make([]cmdutil.ERower, len(ed.erows))
	i := 0
	for _, erow := range ed.erows {
		u[i] = erow
		i++
	}
	return u
}

func (ed *Editor) NewERow(tbStr string, col *ui.Column, rowIndex int) cmdutil.ERower {
	row := col.NewRow(rowIndex)
	erow := NewERow(ed, row, tbStr)

	// add/remove to erows
	ed.erows[row] = erow
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

func (ed *Editor) FindERow(str string) (cmdutil.ERower, bool) {
	//str = cleanTrailingSlash(str)
	for _, erow := range ed.erows {
		tsd := erow.ToolbarSD()
		s1 := tsd.DecodeFirstPart()
		if s1 == str {
			return erow, true
		}
	}
	return nil, false
}

func (ed *Editor) Errorf(f string, a ...interface{}) {
	ed.Error(fmt.Errorf(f, a...))
}
func (ed *Editor) Error(err error) {
	s := "+Errors"
	erow, ok := ed.FindERow(s)
	if !ok {
		col, rowIndex := ed.GoodColRowPlace()
		erow = ed.NewERow(s, col, rowIndex)
	}
	erow.TextAreaAppend(err.Error() + "\n")
	erow.Row().Square.WarpPointer()
}

// Used to run layout toolbar commands.
func (ed *Editor) activeERow() (*ERow, bool) {
	for _, erow := range ed.erows {
		if erow.row.Square.Value(ui.SquareActive) {
			return erow, true
		}
	}
	return nil, false
}

func (ed *Editor) GoodColRowPlace() (*ui.Column, int) {
	col := ed.ui.Layout.Cols.ColumnWithGoodPlaceForNewRow()
	return col, len(col.Rows)
}

func (ed *Editor) onLayoutToolbarSetStr(ev0 xgbutil.EREvent) {
	ev := ev0.(*ui.TextAreaSetStrEvent)
	ed.updateHomeVars(ev)
}
func (ed *Editor) updateHomeVars(ev *ui.TextAreaSetStrEvent) {
	return

	panic("TODO")

	tb := ed.ui.Layout.Toolbar

	// get layout old home vars
	oldVars := getLayoutHomeVars(ev.OldStr)
	log.Print(oldVars)

	// remove all old home vars in all rows
	m := make(map[*ERow]string)
	for _, erow := range ed.erows {
		m[erow] = decodeToolbar(erow)
	}

	// get layout home vars
	vars := getLayoutHomeVars(tb.Str())

	// add vars to home vars
	for i := 0; i < len(vars); i += 2 {
		toolbardata.AppendHomeVar(vars[i], vars[i+1])
	}

	// insert home vars in all rows
	for erow, s := range m {
		erow.row.Toolbar.SetStrClear(s, false, false)
	}
}
func getLayoutHomeVars(str string) []string {
	var vars []string
	tbsd := toolbardata.NewStringData(str)
	for _, part := range tbsd.Parts {
		if len(part.Args) != 1 {
			continue
		}
		str := part.Args[0].Str
		a := strings.Split(str, "=")
		if len(a) != 2 {
			continue
		}
		key, val := a[0], a[1]
		vars = append(vars, key, val)
	}
	return vars
}
func decodeToolbar(erow *ERow) string {
	str := erow.row.Toolbar.Str()
	tbsd := toolbardata.NewStringData(str)
	tbsd.DecodeFirstPart()
	//return toolbardata.

	panic("TODO")

	return ""
}
