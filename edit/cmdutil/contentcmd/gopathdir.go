package contentcmd

import (
	"os"
	"path"
	"strings"

	"github.com/jmigpin/editor/edit/cmdutil"
	"github.com/jmigpin/editor/ui"
)

// Get strings enclosed in quotes, like an import line in a go file, and open the file if found in GOROOT/GOPATH directories.
func goPathDir(ed cmdutil.Editori, row *ui.Row, s string) bool {
	gopath := os.Getenv("GOPATH")
	a := strings.Split(gopath, ":")
	a = append(a, os.Getenv("GOROOT"))
	for _, p := range a {
		p2 := path.Join(p, "src", s)
		_, err := os.Stat(p2)
		if err == nil {
			col := ed.ActiveColumn()
			row, err = ed.FindRowOrCreateInColFromFilepath(p2, col)
			if err == nil {
				row.Square.WarpPointer()
			}
			return true
		}
	}
	return false
}
