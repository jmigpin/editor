package cmdutil

import (
	"os/exec"
	"path"
)

func FilemanagerShortcut(erow ERower) {
	dir := ""
	fp, fi, ok := erow.FileInfo()
	if ok {
		if fi.IsDir() {
			dir = fp
		} else {
			dir = path.Dir(fp)
		}
	} else {
		// try base dir of firstpart
		tsd := erow.ToolbarSD()
		fp, ok := tsd.DecodePart0Arg0()
		if ok {
			dir = path.Dir(fp)
		}
	}
	c := exec.Command("filemanager.sh", dir)
	go c.Run()
}
