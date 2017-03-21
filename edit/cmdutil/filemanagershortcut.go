package cmdutil

import (
	"os/exec"
	"path"
)

func FilemanagerShortcut(erow ERower) {
	dir := ""
	tsd := erow.ToolbarSD()
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
