package edit

import (
	"fmt"

	"github.com/jmigpin/editor/edit/cmdutil"
	"github.com/jmigpin/editor/edit/toolbardata"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/ui/tautil"
)

func ToolbarCmdFromLayout(ed *Editor, ta *ui.TextArea) {
	tsd := toolbardata.NewStringData(ta.Str())
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
	tsd := ed.RowToolbarStringData(row)
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

func layoutToolbarCmd(ed *Editor, ta *ui.TextArea, part *toolbardata.Part) bool {
	p0 := part.Args[0].Trim()
	switch p0 {
	case "Exit":
		ed.Close()
	case "SaveSession":
		cmdutil.SaveSession(ed, part)
	case "OpenSession":
		cmdutil.OpenSession(ed, part)
	case "DeleteSession":
		cmdutil.DeleteSession(ed, part)
	case "ListSessions":
		cmdutil.ListSessions(ed)
	case "NewColumn":
		_ = ed.ui.Layout.Cols.NewColumn()
	case "SaveAllFiles":
		cmdutil.SaveRowsFiles(ed)
	case "ReloadAll":
		cmdutil.ReloadRows(ed)
	case "ReloadAllFiles":
		cmdutil.ReloadRowsFiles(ed)
	case "NewRow":
		row := ed.NewRow(ed.ActiveColumn())
		row.Square.WarpPointer()
	default:
		return false
	}
	return true
}

// returns if cmd was found
func rowToolbarCmd(ed *Editor, row *ui.Row, part *toolbardata.Part) bool {
	p0 := part.Args[0].Trim()
	switch p0 {
	case "Save":
		cmdutil.SaveRowFile(ed, row)
	case "Reload":
		cmdutil.ReloadRow(ed, row)
	case "Close":
		row.Close()
	case "CloseColumn":
		row.Col.Cols.CloseColumnEnsureOne(row.Col)
	case "Find":
		s := part.JoinArgsFromIndex(1).Trim()
		tautil.Find(row.TextArea, s)
	case "GotoLine":
		s := part.JoinArgsFromIndex(1).Trim()
		tautil.GotoLine(row.TextArea, s)
	case "Replace":
		cmdutil.Replace(ed, row, part)
	case "Stop":
		rowCtx.Cancel(row)
	case "ListDir":
		tree, hidden := false, false
		ListDirEd(ed, row, tree, hidden)
	case "ListDirSub":
		tree, hidden := true, false
		ListDirEd(ed, row, tree, hidden)
	case "ListDirHidden":
		tree, hidden := false, true
		ListDirEd(ed, row, tree, hidden)
	default:
		return false
	}
	return true
}
