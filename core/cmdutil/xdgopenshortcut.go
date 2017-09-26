package cmdutil

import "os/exec"

func XdgOpenDirShortcut(ed Editorer) {
	erow, ok := ed.ActiveERow()
	if !ok {
		return
	}

	dir := erow.Dir()
	c := exec.Command("xdg-open", dir)
	go c.Run()
}
