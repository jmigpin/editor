package edit

import (
	"fmt"
	"jmigpin/editor/edit/toolbar"
	"jmigpin/editor/ui"
	"os"
)

func saveRowsFiles(ed *Editor) {
	for _, c := range ed.ui.Layout.Cols.Cols {
		for _, r := range c.Rows {
			saveRowFile2(ed, r, true)
		}
	}
}
func saveRowFile(ed *Editor, row *ui.Row) {
	saveRowFile2(ed, row, false)
}

// Can be tolerant above rows not having filenames
func saveRowFile2(ed *Editor, row *ui.Row, tolerant bool) {
	tsd := toolbar.NewStringData(row.Toolbar.Text())
	filename, ok := tsd.FilenameTag()
	if !ok {
		if !tolerant {
			ed.Error(fmt.Errorf("row has no filename"))
		}
		return
	}

	// best effort to disable/enable filesstates watcher, ignore errors
	_ = ed.fs.Remove(filename)
	defer func() {
		_ = ed.fs.Add(filename)
	}()

	// save
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		ed.Error(err)
		return
	}
	defer f.Close()
	data := []byte(row.TextArea.Text())
	_, err = f.Write(data)
	if err != nil {
		ed.Error(err)
		return
	}

	row.Square.SetDirty(false)
	row.Square.SetCold(false)
}
