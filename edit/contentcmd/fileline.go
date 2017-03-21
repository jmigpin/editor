package contentcmd

import (
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/jmigpin/editor/edit/cmdutil"
)

// Opens filename at line, like in compiler errors <string:int> format.
func fileLine(erow cmdutil.ERower, scmd string) bool {
	// filename
	a := strings.Split(scmd, ":")
	filename := a[0]
	if !path.IsAbs(filename) {
		filepath, fi, ok := erow.FileInfo()
		if ok {
			if fi.IsDir() {
				filename = path.Join(filepath, filename)
			} else {
				filename = path.Join(path.Dir(filepath), filename)
			}
		}
	}
	fi, err := os.Stat(filename)
	if err != nil {
		return false
	}
	if fi.IsDir() {
		return false
	}
	// line number
	num := 0
	if len(a) >= 2 {
		v, err := strconv.ParseUint(a[1], 10, 64)
		if err == nil {
			num = int(v)
		}
	}
	// open
	cmdutil.OpenFileLineAtCol(erow.Editorer(), filename, num, erow.Row().Col)
	return true
}
