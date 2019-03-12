package core

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jmigpin/editor/core/parseutil"
	"github.com/jmigpin/editor/core/toolbarparser"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget/textutil"
)

func RootToolbarCmd(ed *Editor, tb *ui.Toolbar) {
	tbdata := toolbarparser.Parse(tb.Str())
	part, ok := tbdata.PartAtIndex(int(tb.TextCursor.Index()))
	if !ok {
		ed.Errorf("missing part at index")
		return
	}
	if len(part.Args) == 0 {
		ed.Errorf("part at index has no args")
		return
	}

	toolbarCmd(ed, part, nil)
}

//----------

func RowToolbarCmd(erow *ERow) {
	part, ok := erow.TbData.PartAtIndex(int(erow.Row.Toolbar.TextCursor.Index()))
	if !ok {
		erow.Ed.Errorf("missing part at index")
		return
	}
	if len(part.Args) == 0 {
		erow.Ed.Errorf("part at index has no args")
		return
	}

	// first part cmd
	if part == erow.TbData.Parts[0] {
		if !rowFirstPartToolbarCmd(erow, part) {
			erow.Ed.Errorf("no cmd was run")
		}
		return
	}

	toolbarCmd(erow.Ed, part, erow)
}

func rowFirstPartToolbarCmd(erow *ERow, part *toolbarparser.Part) bool {
	a0 := part.Args[0]
	ci := erow.Row.Toolbar.TextCursor.Index()

	// cursor index beyond arg0
	if ci > a0.End {
		return false
	}

	// get path up to cursor index
	a0ci := ci - a0.Pos
	filename := a0.Str()
	i := strings.Index(filename[a0ci:], string(filepath.Separator))
	if i >= 0 {
		filename = filename[:a0ci+i]
	}

	// decode filename
	filename = erow.Ed.HomeVars.Decode(filename)

	// create new row
	info := erow.Ed.ReadERowInfo(filename)
	erow2, err := info.NewERow(erow.Row.PosBelow())
	if err != nil {
		erow.Ed.Error(err)
		return true
	}

	erow2.Flash()

	// set same offset if not dir
	if erow2.Info.IsFileButNotDir() {
		ta := erow.Row.TextArea
		ta2 := erow2.Row.TextArea
		ta2.TextCursor.SetIndex(ta.TextCursor.Index())
		ta2.SetRuneOffset(ta.RuneOffset())
	}

	return true
}

//----------

// erow can be nil (ex: a root toolbar cmd)
func toolbarCmd(ed *Editor, part *toolbarparser.Part, erow *ERow) {
	arg0 := part.Args[0].UnquotedStr()

	rootOnlyCmd := func(fn func()) {
		if erow != nil {
			ed.Errorf("%s:  root toolbar only command", arg0)
			return
		}
		fn()
	}

	currentERow := func() *ERow {
		if erow != nil {
			return erow
		}
		e, ok := ed.ActiveERow()
		if ok {
			return e
		}
		return nil
	}

	rowCmdErr := func(fn func(*ERow) error) {
		e := currentERow()
		if e == nil {
			ed.Errorf("%s: no active row", arg0)
			return
		}
		if err := fn(e); err != nil {
			ed.Errorf("%v: %v", arg0, err)
		}
	}

	rowCmd := func(fn func(*ERow)) {
		rowCmdErr(func(e *ERow) error {
			fn(e)
			return nil
		})
	}

	switch arg0 {
	case "Exit":
		rootOnlyCmd(func() { ed.Close() })

	case "SaveSession":
		rootOnlyCmd(func() { SaveSession(ed, part) })
	case "OpenSession":
		rootOnlyCmd(func() { OpenSession(ed, part) })
	case "DeleteSession":
		rootOnlyCmd(func() { DeleteSession(ed, part) })
	case "ListSessions":
		rootOnlyCmd(func() { ListSessions(ed) })

	case "NewColumn":
		rootOnlyCmd(func() { ed.NewColumn() })
	case "CloseColumn":
		rowCmd(func(e *ERow) { e.Row.Col.Close() })

	case "NewRow":
		rootOnlyCmd(func() { NewRowCmd(ed) })
	case "CloseRow":
		rowCmd(func(e *ERow) { e.Row.Close() })
	case "ReopenRow":
		rootOnlyCmd(func() { ed.RowReopener.Reopen() })
	case "MaximizeRow":
		rowCmd(func(e *ERow) { e.Row.Maximize() })

	case "Save":
		rowCmd(func(e *ERow) { SaveCmd(e.Info) })
	case "SaveAllFiles":
		rootOnlyCmd(func() { SaveAllFilesCmd(ed) })

	case "Reload":
		rowCmd(func(e *ERow) { ReloadCmd(e) })
	case "ReloadAllFiles":
		rootOnlyCmd(func() { ReloadAllFilesCmd(ed) })
	case "ReloadAll":
		rootOnlyCmd(func() { ReloadAllCmd(ed) })

	case "Stop":
		rowCmd(func(e *ERow) { e.Exec.Stop() })

	case "Clear":
		rowCmd(func(e *ERow) { e.Row.TextArea.SetStrClearHistory("") })

	case "Find":
		rowCmdErr(func(e *ERow) error { return FindCmd(e, part) })
	case "Replace":
		rowCmdErr(func(e *ERow) error { return ReplaceCmd(e, part) })
	case "GotoLine":
		rowCmdErr(func(e *ERow) error { return GotoLineCmd(e, part) })
	case "CopyFilePosition":
		rowCmdErr(func(e *ERow) error { return CopyFilePositionCmd(ed, e) })
	case "ToggleRowHBar":
		rowCmdErr(func(e *ERow) error { return ToggleRowHBarCmd(ed, e) })

	case "ListDir":
		rowCmdErr(func(e *ERow) error { return ListDirCmd(e, part) })

	case "XdgOpenDir":
		rowCmdErr(func(e *ERow) error { return XdgOpenDirCmd(e) })
	case "GoRename":
		rowCmdErr(func(e *ERow) error { return GoRenameCmd(e, part) })
	case "GoDebug":
		rowCmdErr(func(e *ERow) error { return GoDebugCmd(e, part) })

	case "ColorTheme":
		colorThemeCmd(ed)
	case "FontTheme":
		fontThemeCmd(ed)
	case "FontRunes":
		fontRunesCmd(ed)

	default:
		// have a plugin handle the cmd
		e := currentERow() // could be nil
		handled := ed.Plugins.RunToolbarCmd(e, part)

		// run external cmd
		if !handled {
			rowCmd(func(e *ERow) { ExternalCmd(e, part) })
		}
	}
}

//----------

func NewRowCmd(ed *Editor) {
	p, err := os.Getwd()
	if err != nil {
		ed.Error(err)
		return
	}

	rowPos := ed.GoodRowPos()

	aerow, ok := ed.ActiveERow()
	if ok {
		// stick with directory if exists, otherwise get base dir
		p2 := aerow.Info.Name()
		if aerow.Info.IsDir() {
			p = p2
		} else {
			p = path.Dir(p2)
		}

		// position after active row
		rowPos = aerow.Row.PosBelow()
	}

	info := ed.ReadERowInfo(p)

	erow := NewERow(ed, info, rowPos)
	erow.Flash()
}

//----------

func SaveCmd(info *ERowInfo) {
	if err := info.SaveFile(); err != nil {
		info.Ed.Error(err)
	}
}
func SaveAllFilesCmd(ed *Editor) {
	for _, info := range ed.ERowInfos {
		if info.IsFileButNotDir() {
			SaveCmd(info)
		}
	}
}

//----------

func ReloadCmd(erow *ERow) {
	erow.Reload()
}
func ReloadAllFilesCmd(ed *Editor) {
	for _, info := range ed.ERowInfos {
		if info.IsFileButNotDir() {
			if err := info.ReloadFile(); err != nil {
				ed.Error(err)
			}
		}
	}
}

func ReloadAllCmd(ed *Editor) {
	// reload all dirs erows
	for _, info := range ed.ERowInfos {
		if info.IsDir() {
			for _, erow := range info.ERows {
				erow.Reload()
			}
		}
	}

	ReloadAllFilesCmd(ed)
}

//----------

func FindCmd(erow *ERow, part *toolbarparser.Part) error {
	args := part.Args[1:]
	if len(args) < 1 {
		return fmt.Errorf("expecting argument")
	}
	var str string
	if len(args) == 1 {
		str = args[0].UnquotedStr()
	} else {
		// join args
		a, b := args[0].Pos, args[len(args)-1].End
		s := part.Data.Str[a:b]
		str = strings.TrimSpace(s)
	}

	found, err := textutil.Find(erow.Row.TextArea.TextEdit, str)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("string not found: %q", str)
	}

	// flash
	tc := erow.Row.TextArea.TextCursor
	a, b := tc.SelectionIndexes()
	erow.MakeRangeVisibleAndFlash(a, b-a)

	return nil
}

//----------

func ReplaceCmd(erow *ERow, part *toolbarparser.Part) error {
	args := part.Args[1:]
	if len(args) != 2 {
		return fmt.Errorf("expecting 2 arguments")
	}

	old, new := args[0].UnquotedStr(), args[1].UnquotedStr()

	replaced, err := textutil.Replace(erow.Row.TextArea.TextEdit, old, new)
	if err != nil {
		return err
	}
	if !replaced {
		return fmt.Errorf("string not replaced: %q", old)
	}
	return nil
}

//----------

func CopyFilePositionCmd(ed *Editor, erow *ERow) error {
	if !erow.Info.IsFileButNotDir() {
		return fmt.Errorf("not a file")
	}

	ta := erow.Row.TextArea
	ci := ta.TextCursor.Index()
	line, col := parseutil.IndexLineColumn(ta.Str()[:ci])

	s := fmt.Sprintf("%v:%v:%v", erow.Info.Name(), line, col)

	ta.SetCPCopy(event.CPIPrimary, s)
	ta.SetCPCopy(event.CPIClipboard, s)

	return nil
}

//----------

func GotoLineCmd(erow *ERow, part *toolbarparser.Part) error {
	args := part.Args[1:]
	if len(args) != 1 {
		return fmt.Errorf("expecting 1 argument")
	}

	line0, err := strconv.ParseUint(args[0].Str(), 10, 64)
	if err != nil {
		return err
	}
	line := int(line0)

	ta := erow.Row.TextArea
	index := parseutil.LineColumnIndex(ta.Str(), line, 0)
	if index < 0 {
		return fmt.Errorf("line not found: %v", line)
	}

	// goto index
	tc := ta.TextCursor
	tc.SetSelectionOff()
	tc.SetIndex(index)

	erow.MakeIndexVisibleAndFlash(index)

	return nil
}

//----------

func XdgOpenDirCmd(erow *ERow) error {
	if erow.Info.IsSpecial() {
		return fmt.Errorf("can't run on special row")
	}

	dir := erow.Info.Dir()
	c := exec.Command("xdg-open", dir)
	if err := c.Start(); err != nil {
		return err
	}
	go func() {
		if err := c.Wait(); err != nil {
			erow.Ed.Error(err)
		}
	}()

	return nil
}

//----------

func colorThemeCmd(ed *Editor) {
	ui.ColorThemeCycler.Cycle(ed.UI.Root)
	ed.UI.Root.MarkNeedsLayoutAndPaint()
}
func fontThemeCmd(ed *Editor) {
	ui.FontThemeCycler.Cycle(ed.UI.Root)
	ed.UI.Root.MarkNeedsLayoutAndPaint()
}

//----------

func fontRunesCmd(ed *Editor) {
	var u string
	for i := 0; i < 15000; {
		start := i
		var w string
		for j := 0; j < 25; j++ {
			w += string(rune(i))
			i++
		}
		u += fmt.Sprintf("%d: %s\n", start, w)
	}
	ed.Messagef("%s", u)
}

//----------

func ToggleRowHBarCmd(ed *Editor, erow *ERow) error {
	erow.Row.ToggleTextAreaXBar()
	return nil
}
