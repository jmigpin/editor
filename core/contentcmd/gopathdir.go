package contentcmd

import (
	"os"
	"path"
	"strings"

	"github.com/jmigpin/editor/core/cmdutil"
)

// Get strings enclosed in quotes, like an import line in a go file, and open the file if found in GOROOT/GOPATH directories.
func goPathDir(erow cmdutil.ERower, s string) bool {
	if s == "" {
		return false
	}

	ed := erow.Ed()

	gopath := os.Getenv("GOPATH")
	a := strings.Split(gopath, ":")
	a = append(a, os.Getenv("GOROOT"))
	for _, p := range a {
		p2 := path.Join(p, "src", s)
		_, err := os.Stat(p2)
		if err == nil {
			erow2, ok := ed.FindERow(p2)
			if !ok {
				col, nextRow := ed.GoodColumnRowPlace()
				erow2 = ed.NewERowBeforeRow(p2, col, nextRow)
				err = erow2.LoadContentClear()
				if err != nil {
					ed.Error(err)
					return true
				}
			}
			erow2.Row().WarpPointer()
			return true
		}
	}
	return false
}
