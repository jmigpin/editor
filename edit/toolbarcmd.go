package edit

import (
	"fmt"

	"github.com/jmigpin/editor/edit/toolbar"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/ui/tautil"
)

func ToolbarCmdFromLayout(ed *Editor, ta *ui.TextArea) {
	tsd := toolbar.NewStringData(ta.Str())
	part, ok := tsd.GetPartAtIndex(ta.CursorIndex())
	if !ok {
		return
	}
	ok = layoutToolbarCmd(ed, ta, part)
	if ok {
		return
	}
	// try running row command
	row, ok := ed.activeRow()
	if ok {
		ok := rowToolbarCmd(ed, row, part)
		if ok {
			return
		}
	}
	// TODO: consider running external command in new row
	ed.Error(fmt.Errorf("unknown layout command: %v", part.Str))
}

func ToolbarCmdFromRow(ed *Editor, row *ui.Row) {
	tsd := ed.rowToolbarStringData(row)
	ta := row.Toolbar
	part, ok := tsd.GetPartAtIndex(ta.CursorIndex())
	if !ok {
		return
	}
	ok = rowToolbarCmd(ed, row, part)
	if ok {
		return
	}
	// external command
	cmd := part.JoinArgs().Trim()
	ToolbarCmdExternalForRow(ed, row, cmd)
}

func layoutToolbarCmd(ed *Editor, ta *ui.TextArea, part *toolbar.Part) bool {
	p0 := part.Args[0].Trim()
	switch p0 {
	case "Exit":
		ed.Close()
	case "SaveSession":
		saveSession(ed, part)
	case "OpenSession":
		openSession(ed, part)
	case "DeleteSession":
		deleteSession(ed, part)
	case "ListSessions":
		listSessions(ed)
	case "NewColumn":
		_ = ed.ui.Layout.Cols.NewColumn()
	case "SaveAll":
		saveRowsFiles(ed)
	case "ReloadAll":
		reloadRows(ed)
	case "ReloadAllFiles":
		reloadRowsFiles(ed)
	case "NewRow":
		var col *ui.Column
		arow, ok := ed.activeRow()
		if ok {
			col = arow.Col
		} else {
			col = ed.ui.Layout.Cols.LastColumnOrNew()
		}
		row := col.NewRow()
		row.Square.WarpPointer()
	default:
		return false
	}
	return true
}

// returns if cmd was found
func rowToolbarCmd(ed *Editor, row *ui.Row, part *toolbar.Part) bool {
	p0 := part.Args[0].Trim()
	switch p0 {
	case "Save":
		saveRowFile(ed, row)
	case "Reload":
		reloadRow(ed, row)
	case "Close":
		row.Close()
	case "CloseColumn":
		row.Col.Cols.RemoveColumnEnsureOne(row.Col)
	case "Find":
		s := part.JoinArgsFromIndex(1).Trim()
		tautil.Find(row.TextArea, s)
	case "GotoLine":
		s := part.JoinArgsFromIndex(1).Trim()
		tautil.GotoLine(row.TextArea, s)
	case "Cut":
		tautil.Cut(row.TextArea)
	case "Copy":
		tautil.Copy(row.TextArea)
	case "Paste":
		tautil.Paste(row.TextArea)
	case "Replace":
		a := part.Args[1:]
		if len(a) != 2 {
			ed.Error(fmt.Errorf("replace: expecting 2 arguments"))
		} else {
			old, new := a[0].Trim(), a[1].Trim()
			tautil.Replace(row.TextArea, old, new)
		}
	case "Stop":
		rowCtx.Cancel(row)
	case "ListDirTree":
		tree, hidden, err := parseTwoOptionalTrueFalseArgs(part)
		if err != nil {
			ed.Error(err)
		} else {
			ListDirTreeEd(ed, row, tree, hidden)
		}
	case "ListDirSubTree":
		tree, hidden := true, false
		ListDirTreeEd(ed, row, tree, hidden)
	default:
		return false
	}
	return true
}

func parseTwoOptionalTrueFalseArgs(part *toolbar.Part) (bool, bool, error) {
	parseTrueFalse := func(s string) (bool, error) {
		if s == "true" {
			return true, nil
		} else if s == "false" {
			return false, nil
		}
		return false, fmt.Errorf("expecting true/false argument")
	}
	// parse args
	n := 2
	if len(part.Args)-1 > n { // minus first arg
		return false, false, fmt.Errorf("expecting at most 2 arguments")
	}
	b := make([]bool, n)
	for i := 1; i < len(part.Args); i++ {
		v, err := parseTrueFalse(part.Args[i].Trim())
		if err != nil {
			return false, false, err
		}
		b[i-1] = v
	}
	return b[0], b[1], nil
}
