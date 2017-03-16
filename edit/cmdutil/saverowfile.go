package cmdutil

import (
	"os"

	"github.com/jmigpin/editor/ui"
)

func SaveRowsFiles(ed Editorer) {
	for _, c := range ed.UI().Layout.Cols.Cols {
		for _, r := range c.Rows {
			saveRowFile2(ed, r, true)
		}
	}
}
func SaveRowFile(ed Editorer, row *ui.Row) {
	saveRowFile2(ed, row, false)
}

func saveRowFile2(ed Editorer, row *ui.Row, tolerant bool) {
	tsd := ed.RowToolbarStringData(row)
	// file might not exist yet, so getting from filepath
	filename := tsd.FirstPartFilepath()

	// best effort to disable/enable file watcher, ignore errors
	_ = ed.FilesWatcherRemove(filename)
	defer func() {
		_ = ed.FilesWatcherAdd(filename)
	}()

	// save
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		ed.Error(err)
		return
	}
	defer f.Close()
	data := []byte(row.TextArea.Str())
	_, err = f.Write(data)
	if err != nil {
		ed.Error(err)
		return
	}

	row.Square.SetDirty(false)
	row.Square.SetCold(false)
}
