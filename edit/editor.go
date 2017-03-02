package edit

import (
	"flag"
	"fmt"
	"image"
	"log"
	"net/url"
	"os"
	"runtime/pprof"
	"strings"

	"github.com/jmigpin/editor/edit/cmdutil"
	"github.com/jmigpin/editor/edit/cmdutil/contentcmd"
	"github.com/jmigpin/editor/edit/toolbardata"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/xutil/dragndrop"

	"github.com/BurntSushi/xgb/xproto"
	"github.com/fsnotify/fsnotify"
)

type Editor struct {
	ui *ui.UI
	fs *FilesState // TODO: rename fileswatcher
}

func NewEditor() (*Editor, error) {
	ed := &Editor{}

	ui0, err := ui.NewUI()
	if err != nil {
		return nil, err
	}
	ed.ui = ui0
	ed.ui.OnEvent = ed.onUIEvent

	fs, err := NewFilesState()
	if err != nil {
		return nil, err
	}
	ed.fs = fs
	ed.fs.OnError = ed.Error
	ed.fs.OnEvent = ed.onFSEvent

	// set up layout toolbar
	ta := ed.ui.Layout.Toolbar
	s := "Exit | ListSessions | NewColumn | NewRow"
	ta.ClearStr(s, false)

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
	go ed.fs.EventLoop()
	ed.ui.EventLoop()

	return ed, nil
}

func (ed *Editor) UI() *ui.UI {
	return ed.ui
}
func (ed *Editor) FilesWatcherAdd(filename string) error {
	return ed.fs.Add(filename)
}
func (ed *Editor) FilesWatcherRemove(filename string) error {
	return ed.fs.Remove(filename)
}

func (ed *Editor) Close() {
	ed.fs.Close()
	ed.ui.Close()
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

func (ed *Editor) findRow(s string) (*ui.Row, bool) {
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
func (ed *Editor) FindRowOrCreate(name string) *ui.Row {
	row, ok := ed.findRow(name)
	if ok {
		return row
	}
	// new row
	col := ed.ActiveColumn()
	row = col.NewRow()
	row.Toolbar.ClearStr(name, false)
	return row
}
func (ed *Editor) FindRowOrCreateInColFromFilepath(filepath string, col *ui.Column) (*ui.Row, error) {
	row, ok := ed.findRow(filepath)
	if ok {
		return row, nil
	}
	// new row
	content, err := filepathContent(filepath)
	if err != nil {
		return nil, err
	}
	row = col.NewRow()
	p2 := toolbardata.InsertHomeTilde(filepath)
	row.Toolbar.ClearStr(p2+" | Reload", false)
	row.TextArea.ClearStr(content, false)
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

//func (ed *Editor) onSignal(sig os.Signal) {
//fmt.Printf("signal: %v\n", sig)
//}

func (ed *Editor) Error(err error) {
	row := ed.FindRowOrCreate("+Errors")
	ta := row.TextArea
	// append
	ta.ClearStr(ta.Str()+err.Error()+"\n", true)
}

func (ed *Editor) onUIEvent(ev ui.Event) {
	switch ev0 := ev.(type) {
	case error:
		ed.Error(ev0)
	case *ui.TextAreaCmdEvent:
		ed.onTextAreaCmd(ev0)
	case *ui.TextAreaSetTextEvent:
		ed.onTextAreaSetText(ev0)
	case *ui.RowKeyPressEvent:
		ed.onRowKeyPress(ev0)
	case *ui.RowCloseEvent:
		rowCtx.Cancel(ev0.Row)
		ed.updateFileStates()
	case *ui.ColumnDndPositionEvent:
		ed.onColumnDndPosition(ev0)
	case *ui.ColumnDndDropEvent:
		ed.onColumnDndDrop(ev0)
	default:
		fmt.Printf("editor unhandled event: %v\n", ev)
	}
}
func (ed *Editor) onTextAreaCmd(ev *ui.TextAreaCmdEvent) {
	ta := ev.TextArea
	switch t0 := ta.Data.(type) {
	case *ui.Toolbar:
		switch t1 := t0.Data.(type) {
		case *ui.Layout:
			ToolbarCmdFromLayout(ed, ta)
		case *ui.Row:
			ToolbarCmdFromRow(ed, t1)
		}
	case *ui.Row:
		switch ta {
		case t0.TextArea:
			contentcmd.Cmd(ed, t0)
		}
	}
}
func (ed *Editor) onTextAreaSetText(ev *ui.TextAreaSetTextEvent) {
	ta := ev.TextArea
	switch t0 := ta.Data.(type) {
	case *ui.Toolbar:
		switch t1 := t0.Data.(type) {
		case *ui.Row:
			tsd := ed.RowToolbarStringData(t1)
			_, ok := tsd.FirstPartFilename()
			if ok {
				ed.updateFileStates()
			}
		}
	case *ui.Row:
		switch ta {
		case t0.TextArea:
			// set as dirty only if it has a filename
			tsd := ed.RowToolbarStringData(t0)
			_, ok := tsd.FirstPartFilename()
			if ok {
				t0.Square.SetDirty(true)
			}
		}
	}
}
func (ed *Editor) onRowKeyPress(ev *ui.RowKeyPressEvent) {
	fks := ev.Key.FirstKeysym()
	m := ev.Key.Modifiers
	if m.Control() && fks == 's' {
		cmdutil.SaveRowFile(ed, ev.Row)
		return
	}
	if m.Control() && m.Shift() && fks == 'f' {
		cmdutil.FilemanagerShortcut(ed, ev.Row)
		return
	}
	if m.Control() && fks == 'f' {
		cmdutil.QuickFindShortcut(ed, ev.Row)
		return
	}
}
func (ed *Editor) onColumnDndPosition(ev *ui.ColumnDndPositionEvent) {
	// supported types
	ok := false
	types := []xproto.Atom{dragndrop.DropTypeAtoms.TextURLList}
	for _, t := range types {
		if ev.Event.SupportsType(t) {
			ok = true
			break
		}
	}
	if ok {
		// TODO: if ctrl is pressed, set to XdndActionLink
		// reply accept with action
		action := dragndrop.DndAtoms.XdndActionCopy
		ev.Event.ReplyAccept(action)
	}
}
func (ed *Editor) onColumnDndDrop(ev *ui.ColumnDndDropEvent) {
	data, err := ev.Event.RequestData(dragndrop.DropTypeAtoms.TextURLList)
	if err != nil {
		ev.Event.ReplyDeny()
		ed.Error(err)
		return
	}
	urls, err := parseAsTextURLList(data)
	if err != nil {
		ev.Event.ReplyDeny()
		ed.Error(err)
		return
	}
	ed.handleColumnDroppedURLs(ev.Column, ev.Point, urls)
	ev.Event.ReplyAccepted()
}
func parseAsTextURLList(data []byte) ([]*url.URL, error) {
	s := string(data)
	entries := strings.Split(s, "\n")
	var urls []*url.URL
	for _, e := range entries {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		u, err := url.Parse(e)
		if err != nil {
			return nil, err
		}
		urls = append(urls, u)
	}
	return urls, nil
}
func (ed *Editor) handleColumnDroppedURLs(col *ui.Column, p *image.Point, urls []*url.URL) {
	for _, u := range urls {
		if u.Scheme == "file" {
			row, err := ed.FindRowOrCreateInColFromFilepath(u.Path, col)
			if err != nil {
				ed.Error(err)
				continue
			}
			col.Cols.MoveRowToPoint(row, p)
		}
	}
}

func (ed *Editor) updateFileStates() {
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
	ed.fs.SetFiles(u)
}
func (ed *Editor) onFSEvent(ev fsnotify.Event) {
	evs := fsnotify.Create | fsnotify.Write | fsnotify.Remove | fsnotify.Rename
	if ev.Op&evs > 0 {
		row, ok := ed.findRow(ev.Name)
		if ok {
			row.Square.SetCold(true)
		}
		return
	} else if ev.Op&fsnotify.Chmod > 0 {
		return
	}
	ed.Error(fmt.Errorf("unhandled fs event: %v", ev))
	// The window might not have focus, and no expose event will happen
	// Since the fsevents are async, a request is done to ensure a draw
	ed.ui.RequestTreePaint()
}
