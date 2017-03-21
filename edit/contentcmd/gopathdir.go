package contentcmd

import (
	"os"
	"path"
	"strings"

	"github.com/jmigpin/editor/edit/cmdutil"
)

// Get strings enclosed in quotes, like an import line in a go file, and open the file if found in GOROOT/GOPATH directories.
func goPathDir(erow cmdutil.ERower, s string) bool {
	if s == "" {
		return false
	}
	gopath := os.Getenv("GOPATH")
	a := strings.Split(gopath, ":")
	a = append(a, os.Getenv("GOROOT"))
	for _, p := range a {
		p2 := path.Join(p, "src", s)
		_, err := os.Stat(p2)
		if err == nil {
			ed := erow.Editorer()
			col := ed.ActiveColumn()
			erow := ed.FindERowOrCreate(p2, col)
			err = erow.LoadContentClear()
			if err == nil {
				erow.Row().Square.WarpPointer()
			}
			return true
		}
	}
	return false
}
