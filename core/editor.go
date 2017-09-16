package core

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/golang/freetype/truetype"
	"github.com/jmigpin/editor/core/cmdutil"
	"github.com/jmigpin/editor/drawutil2"
	"github.com/jmigpin/editor/drawutil2/loopers"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/xgbutil"
	"github.com/jmigpin/editor/xgbutil/wmprotocols"
)

type Editor struct {
	ui        *ui.UI
	fw        *FilesWatcher
	reopenRow *cmdutil.ReopenRow
	erows     map[*ui.Row]*ERow
	close     chan struct{}
}

func NewEditor(opt *Options) (*Editor, error) {
	ed := &Editor{
		erows: make(map[*ui.Row]*ERow),
		close: make(chan struct{}),
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

	if opt.AcmeColors {
		ui.AcmeColors()
	}
	ui.SetScrollbarWidth(opt.ScrollbarWidth)

	loopers.WrapLineRune = rune(opt.WrapLineRune)
	drawutil2.TabWidth = opt.TabWidth

	// close editor when the window is deleted
	ed.ui.EvReg.Add(wmprotocols.DeleteWindowEventId,
		&xgbutil.ERCallback{func(ev0 interface{}) {
			ed.Close()
		}})
	// possible x errors
	ed.ui.EvReg.Add(xgbutil.XErrorEventId,
		&xgbutil.ERCallback{func(ev interface{}) {
			ed.Errorf("xerror: %v", ev)
		}})

	// setup drop support (files, dirs, ...) from other applications
	cmdutil.SetupDragNDrop(ed)

	// set up layout toolbar
	s := "Exit | ListSessions | NewColumn | NewRow | ReopenRow | RowDirectory | Reload | "
	ed.ui.Layout.Toolbar.SetStrClear(s, true, true)
	// execute commands on layout toolbar
	ed.ui.Layout.Toolbar.EvReg.Add(ui.TextAreaCmdEventId,
		&xgbutil.ERCallback{func(ev interface{}) {
			ToolbarCmdFromLayout(ed, ed.ui.Layout)
		}})
	cmdutil.SetupLayoutHomeVars(ed)

	// files watcher for visual feedback when files change
	fw, err := NewFilesWatcher(ed)
	if err != nil {
		return nil, err
	}
	ed.fw = fw

	// cmd line filenames to open
	args := flag.Args()
	if len(args) > 0 {
		col, _ := ed.ui.Layout.Cols.FirstChildColumn()
		for _, s := range args {
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
	} else {
		// start with 2 colums and a current directory row on 2nd column
		cols := ed.ui.Layout.Cols
		_ = cols.NewColumn()
		col, ok := cols.LastChildColumn()
		if ok {
			cmdutil.OpenDirectoryRow(ed, ".", col, nil)
		}
	}

	go ed.fw.EventLoop()
	ed.eventLoop() // blocks

	return ed, nil
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
	ed.fw.Close()
	close(ed.close)
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

func (ed *Editor) NewERowBeforeRow(tbStr string, col *ui.Column, nextRow *ui.Row) cmdutil.ERower {
	row := col.NewRowBefore(nextRow)
	erow := NewERow(ed, row, tbStr)

	// add/remove to erows
	ed.erows[row] = erow
	row.EvReg.Add(ui.RowCloseEventId,
		&xgbutil.ERCallback{func(ev0 interface{}) {
			delete(ed.erows, row)
		}})

	// key shortcuts
	row.EvReg.Add(ui.RowKeyPressEventId,
		&xgbutil.ERCallback{ed.onRowKeyPress})
	return erow
}
func (ed *Editor) onRowKeyPress(ev0 interface{}) {
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
	for _, erow := range ed.erows {
		s1 := erow.DecodedPart0Arg0()
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
		col, nextRow := ed.GoodColumnRowPlace()
		erow = ed.NewERowBeforeRow(s, col, nextRow)
	}
	erow.TextAreaAppendAsync(err.Error() + "\n")
	//erow.Row().WarpPointer()
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

func (ed *Editor) GoodColumnRowPlace() (*ui.Column, *ui.Row) {
	col := ed.ui.Layout.Cols.ColumnWithGoodPlaceForNewRow()
	return col, nil // last position
}

func (ed *Editor) IsSpecialName(s string) bool {
	return len(s) > 0 && s[0] == '+'
}

func (ed *Editor) eventLoop() {
	defer ed.ui.Close()

	var lastPaint time.Time
	paintIfNeeded := func() {
		lastPaint = time.Now()
		ed.ui.PaintIfNeeded()
	}

	for {
	selectStart:
		select {
		case <-ed.close:
			goto forEnd

		case ev, ok := <-ed.ui.Events:
			if !ok {
				goto forEnd
			}
			switch ev2 := ev.(type) {
			case xgb.Event:

				// bypass quick motionnotify events
				// FIXME: can bypass a motion segment if last event is not motion
				if len(ed.ui.Events) > 1 {
					_, ok := ev2.(xproto.MotionNotifyEvent)
					if ok {
						goto selectStart
					}
				}

				eid := xgbutil.XgbEventId(ev2)
				ed.ui.EvReg.Emit(eid, ev2)
			case xgb.Error:
				ed.ui.EvReg.Emit(xgbutil.XErrorEventId, ev2)
			case int:
				ed.ui.EvReg.Emit(ev2, nil)
			case *xgbutil.EREventData:
				ed.ui.EvReg.Emit(ev2.EventId, ev2.Event)
			default:
				log.Printf("unhandled event type: %v", ev)
			}
		}

		if len(ed.ui.Events) == 0 {
			paintIfNeeded()
		} else {
			// ensure a paint at x frames per second
			d := time.Now().Sub(lastPaint)
			if d > time.Second/30 {
				paintIfNeeded()
			}
		}
	}
forEnd:
}

type Options struct {
	FontFilename   string
	FontSize       float64
	DPI            float64
	ScrollbarWidth int
	AcmeColors     bool
	WrapLineRune   int
	TabWidth       int
}
