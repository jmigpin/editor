package contentcmd

import (
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/jmigpin/editor/core/cmdutil"
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
		p = path.Join(erow.Dir(), p)
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
		col, nextRow := ed.GoodColumnRowPlace()
		erow2 = ed.NewERowBeforeRow(p, col, nextRow)
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
	warped := false
	if haveNum {
		ok = tautil.GotoLineNum(erow2.Row().TextArea, num)
		if ok {
			warped = true
		}
	}

	if !warped {
		erow2.Row().WarpPointer()
	}
	return true
}
