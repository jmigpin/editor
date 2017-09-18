package cmdutil

import (
	"os/exec"
	"path"
)

func XdgOpenDirShortcut(ed Editorer) {
	erow, ok := ed.ActiveERow()
	if !ok {
		return
	}

	dir := ""

	// get directory from row
	fp, fi, err := erow.FileInfo()
	if err == nil {
		if fi.IsDir() {
			dir = fp
		} else {
			dir = path.Dir(fp)
		}
	} else {
		// try base dir of part0
		fp := erow.DecodedPart0Arg0()
		dir = path.Dir(fp) // fp=="" gives "."
	}

	c := exec.Command("xdg-open", dir)
	go c.Run()
}
