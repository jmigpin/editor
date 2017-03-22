package edit

import (
	"fmt"

	"github.com/jmigpin/editor/edit/cmdutil"
	"github.com/jmigpin/editor/edit/toolbardata"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/ui/tautil"
)

func ToolbarCmdFromLayout(ed *Editor, layout *ui.Layout) {
	ta := layout.Toolbar.TextArea
	tsd := toolbardata.NewStringData(ta.Str())
	part, ok := tsd.GetPartAtIndex(ta.CursorIndex())
	if !ok {
		return
	}
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
		col, rowIndex := ed.GoodColRowPlace()
		erow := ed.NewERow("", col, rowIndex)
		erow.Row().Square.WarpPointer()
	case "ReopenRow":
		erow, ok := ed.reopenRow.Reopen()
		if ok {
			erow.Row().Square.WarpPointer()
		}
	case "FileManager":
		erow, ok := ed.activeERow()
		if ok {
			cmdutil.FilemanagerShortcut(erow)
		}
	case "RowDirectory":
		erow, ok := ed.activeERow()
		if ok {
			cmdutil.OpenRowDirectory(erow)
		}
	default:
		// try running row command
		erow, ok := ed.activeERow()
		if ok {
			ok := rowToolbarCmd(erow, part)
			if ok {
				return
			}
		}
		// TODO: consider running external command in new row
		err := fmt.Errorf("unknown layout command: %v", part.Str)
		ed.Error(err)
	}
}

func ToolbarCmdFromRow(erow *ERow) {
	tsd := erow.ToolbarSD()
	ta := erow.Row().Toolbar
	part, ok := tsd.GetPartAtIndex(ta.CursorIndex())
	if !ok {
		return
	}
	ok = rowToolbarCmd(erow, part)
	if ok {
		return
	}
	// external command
	cmd := part.JoinArgs().Trim()
	cmdutil.ExternalCmd(erow, cmd)
}

// Returns true if cmd was handled.
func rowToolbarCmd(erow *ERow, part *toolbardata.Part) bool {
	row := erow.Row()
	p0 := part.Args[0].Trim()
	switch p0 {
	case "Save":
		cmdutil.SaveRowFile(erow)
	case "Reload":
		cmdutil.ReloadRow(erow)
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
		cmdutil.Replace(erow, part)
	case "Stop":
		cmdutil.RowCtxCancel(row)
	case "ListDir":
		tree, hidden := false, false
		cmdutil.ListDirEd(erow, tree, hidden)
	case "ListDirSub":
		tree, hidden := true, false
		cmdutil.ListDirEd(erow, tree, hidden)
	case "ListDirHidden":
		tree, hidden := false, true
		cmdutil.ListDirEd(erow, tree, hidden)
	default:
		return false
	}
	return true
}
