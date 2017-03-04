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
	"github.com/howeyc/fsnotify"
)

type Editor struct {
	ui *ui.UI
	fw *FilesWatcher
}

func NewEditor() (*Editor, error) {
	ed := &Editor{}

	ui0, err := ui.NewUI()
	if err != nil {
		return nil, err
	}
	ed.ui = ui0
	ed.ui.OnEvent = ed.onUIEvent

	fw, err := NewFilesWatcher()
	if err != nil {
		return nil, err
	}
	ed.fw = fw
	ed.fw.OnError = ed.Error
	ed.fw.OnEvent = ed.onFWEvent

	// set up layout toolbar
	ta := ed.ui.Layout.Toolbar
	s := "Exit | ListSessions | NewColumn | NewRow"
	ta.SetStrClear(s, true, true)

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
	ed.ui.EventLoop()

	return ed, nil
}

func (ed *Editor) UI() *ui.UI {
	return ed.ui
}
func (ed *Editor) FilesWatcherAdd(filename string) error {
	return ed.fw.Add(filename)
}
func (ed *Editor) FilesWatcherRemove(filename string) error {
	return ed.fw.Remove(filename)
}

func (ed *Editor) Close() {
	ed.fw.Close()
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
	row.Toolbar.SetStrClear(name, true, true)
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

//func (ed *Editor) onSignal(sig os.Signal) {
//fmt.Printf("signal: %v\n", sig)
//}

func (ed *Editor) Error(err error) {
	row := ed.FindRowOrCreate("+Errors")
	ta := row.TextArea
	// append
	a := ta.Str() + err.Error() + "\n"
	ta.SetStrClear(a, false, true)
}
func (ed *Editor) IsSpecialRowName(name string) bool {
	return name[0] == '+'
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
		ed.updateFilesWatcher()
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
			_ = t1
			// in case the filename was changed
			ed.updateFilesWatcher()
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
		cmdutil.FindShortcut(ed, ev.Row)
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
	row, ok := ed.findRow(ev.Name)
	if ok {
		row.Square.SetCold(true)
		// this func was called async, need to request tree paint
		ed.ui.RequestTreePaint()
	}
	// always update
	ed.updateFilesWatcher()
}
