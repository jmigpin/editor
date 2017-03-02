package contentcmd

import (
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/jmigpin/editor/edit/cmdutil"
	"github.com/jmigpin/editor/ui"
)

// Opens filename at line, like in compiler errors <string:int> format.
func fileLine(ed cmdutil.Editori, row *ui.Row, scmd string) bool {
	// filename
	a := strings.Split(scmd, ":")
	filename := a[0]
	if !path.IsAbs(filename) {
		tsd := ed.RowToolbarStringData(row)
		d, ok := tsd.FirstPartDirectory()
		if ok {
			filename = path.Join(d, filename)
		} else {
			f, ok := tsd.FirstPartFilename()
			if ok {
				filename = path.Join(path.Dir(f), filename)
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
	cmdutil.OpenFileLineAtCol(ed, filename, num, row.Col)
	return true
}
