package cmdutil

import "os/exec"

func XdgOpenDirShortcut(ed Editorer) {
	erow, ok := ed.ActiveERower()
	if !ok {
		return
	}

	dir := erow.Dir()
	c := exec.Command("xdg-open", dir)
	go c.Run()
}
