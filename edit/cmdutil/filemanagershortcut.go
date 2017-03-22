package cmdutil

import (
	"os/exec"
	"path"
)

func FilemanagerShortcut(erow ERower) {
	tsd := erow.ToolbarSD()
	fp := tsd.FirstPartFilepath()
	dir := path.Dir(fp)
	c := exec.Command("filemanager.sh", dir)
	go c.Run()
}
