package edit

import (
	"jmigpin/editor/edit/toolbar"
	"jmigpin/editor/ui"
	"os/exec"
	"path"
)

func filemanagerShortcut(row *ui.Row) {
	dir := ""
	tsd := toolbar.NewStringData(row.Toolbar.Text())
	dt, ok := tsd.DirectoryTag()
	if ok {
		dir = dt
	} else {
		ft, ok := tsd.FilenameTag()
		if ok {
			dir = path.Dir(ft)
		}
	}
	c := exec.Command("filemanager.sh", dir)
	go c.Run()
}
