package contentcmd

import (
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/jmigpin/editor/edit/cmdutil"
	"github.com/jmigpin/editor/ui/tautil"
)

// Opens filename. Detects <string:int> format (compiler errors), and goes to line.
func file(erow cmdutil.ERower, str string) bool {
	a := strings.Split(str, ":")
	p := a[0]
	if p == "" {
		return false
	}
	if !path.IsAbs(p) {
		filepath, fi, ok := erow.FileInfo()
		if ok {
			if fi.IsDir() {
				p = path.Join(filepath, p)
			} else {
				p = path.Join(path.Dir(filepath), p)
			}
		}
	}

	fi, err := os.Stat(p)
	if err != nil {
		return false
	}
	if fi.IsDir() {
		return false
	}

	ed := erow.Ed()
	erow2, ok := ed.FindERow(p)
	if !ok {
		col, rowIndex := ed.GoodColRowPlace()
		erow2 = ed.NewERow(p, col, rowIndex)
		err := erow2.LoadContentClear()
		if err != nil {
			ed.Error(err)
			return true
		}
	}

	// line number
	haveNum := false
	num := 0
	if len(a) >= 2 {
		v, err := strconv.ParseUint(a[1], 10, 64)
		if err == nil {
			haveNum = true
			num = int(v)
		}
	}

	// don't search/touch the indexes if the line is not set
	if !haveNum {
		erow2.Row().Square.WarpPointer()
		return true
	}

	ok = tautil.GotoLineNum(erow2.Row().TextArea, num)
	if !ok {
		erow2.Row().Square.WarpPointer()
	}

	return true
}
