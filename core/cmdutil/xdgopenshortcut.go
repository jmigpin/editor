package cmdutil

import "os/exec"

func XdgOpenDirShortcut(ed Editorer, erow ERower) {
	dir := erow.Dir()
	c := exec.Command("xdg-open", dir)
	go c.Run()
}
