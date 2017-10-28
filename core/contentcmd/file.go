package contentcmd

import (
	"strconv"
	"strings"

	"github.com/jmigpin/editor/core/cmdutil"
)

// Opens filename.
// Detects compiler errors format <string(:int)?(:int?)>, and goes to line/column.
func file(erow cmdutil.ERower, str string) bool {
	a := strings.Split(str, ":")

	// filename
	p := a[0]
	if p == "" {
		return false
	}
	filename, fi, ok := findFileinfo(erow, p)
	if !ok || fi.IsDir() {
		return false
	}

	// line number
	line := 0
	if len(a) >= 2 {
		v, err := strconv.ParseUint(a[1], 10, 64)
		if err == nil {
			line = int(v)
		}
	}

	// column number
	column := 0
	if len(a) >= 3 {
		v, err := strconv.ParseUint(a[2], 10, 64)
		if err == nil {
			column = int(v)
		}
	}

	// erow
	ed := erow.Ed()
	erow2, ok := ed.FindERower(filename)
	if !ok {
		col, nextRow := ed.GoodColumnRowPlace()
		erow2 = ed.NewERowerBeforeRow(filename, col, nextRow)
		err := erow2.LoadContentClear()
		if err != nil {
			ed.Error(err)
			return true
		}
	}

	if line == 0 && column == 0 {
		erow2.Row().WarpPointer()
	} else {
		cmdutil.GotoLineColumnInTextArea(erow2.Row(), line, column)
	}

	return true
}
