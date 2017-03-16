package cmdutil

import (
	"os/exec"
	"path"

	"github.com/jmigpin/editor/ui"
)

func FilemanagerShortcut(ed Editorer, row *ui.Row) {
	dir := ""
	tsd := ed.RowToolbarStringData(row)
	d, ok := tsd.FirstPartDirectory()
	if ok {
		dir = d
	} else {
		f, ok := tsd.FirstPartFilename()
		if ok {
			dir = path.Dir(f)
		}
	}
	c := exec.Command("filemanager.sh", dir)
	go c.Run()
}
