package contentcmd

import (
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/jmigpin/editor/core/cmdutil"
)

// Opens filename. Detects compiler errors format <string(:int)?(:int?)>, and goes to line/column.
func file(erow cmdutil.ERower, str string) bool {
	a := strings.Split(str, ":")

	// filename
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
	filename := p

	// line number
	line := 0
	if len(a) >= 2 {
		v, err := strconv.ParseUint(a[1], 10, 64)
		if err == nil {
			line = int(v) - 1
		}
	}

	// column number
	column := 0
	if len(a) >= 3 {
		v, err := strconv.ParseUint(a[2], 10, 64)
		if err == nil {
			column = int(v) - 1
		}
	}

	// erow
	ed := erow.Ed()
	erow2, ok := ed.FindERow(filename)
	if !ok {
		col, nextRow := ed.GoodColumnRowPlace()
		erow2 = ed.NewERowBeforeRow(filename, col, nextRow)
		err := erow2.LoadContentClear()
		if err != nil {
			ed.Error(err)
			return true
		}
	}

	cmdutil.GotoLineColumnInTextArea(erow2.Row().TextArea, line, column)

	return true
}
