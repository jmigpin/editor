package contentcmd

import (
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/jmigpin/editor/core/cmdutil"
	"github.com/jmigpin/editor/ui/tautil"
)

const FilenameStopRunes = "\"'`&=:<>[]"

// Opens filename, including directories.
// Detects compiler errors format <string(:int)?(:int?)>, and goes to line/column.
func filename(erow cmdutil.ERower) bool {
	ta := erow.Row().TextArea

	var str string
	if ta.SelectionOn() {
		a, b := tautil.SelectionStringIndexes(ta)
		str = ta.Str()[a:b]
	} else {
		isStop := StopOnSpaceAndRunesFn(FilenameStopRunes)
		l, r := expandLeftRightStop(ta.Str(), ta.CursorIndex(), isStop)
		str = ta.Str()[l:r]

		// get path up to cursor index to allow opening previous directories
		ci := ta.CursorIndex() - l
		i := strings.Index(str[ci:], string(filepath.Separator))
		if i >= 0 {
			// if there is line/column (parse later), the str will be set and  the full filename is considered
			str = str[:ci+i]

			//// line/column can only parse from here
			//r = l + ci + i
			//str = ta.Str()[l:r]
		}

		// expand string to get line/column
		// line
		if r < len(ta.Str()) && ta.Str()[r] == ':' {
			r2 := expandRightStop(ta.Str(), r+1, NotStop(unicode.IsNumber))
			str = ta.Str()[l:r2]

			// column
			if r2 < len(ta.Str()) && ta.Str()[r2] == ':' {
				r3 := expandRightStop(ta.Str(), r2+1, NotStop(unicode.IsNumber))
				str = ta.Str()[l:r3]
			}
		}
	}

	a := strings.Split(str, ":")

	// filename
	if len(a) == 0 {
		return false
	}
	if a[0] == "" {
		return false
	}
	filename, fi, ok := findFileinfo(erow, a[0])
	if !ok {
		return false
	}

	// line and column
	var line, column int = 0, 0 // if existent, both are in [1,...)
	if fi.Mode().IsRegular() {
		if len(a) > 1 {
			// line
			v, err := strconv.ParseUint(a[1], 10, 64)
			if err == nil {
				line = int(v)
			}
			// column
			if len(a) > 2 {
				v, err := strconv.ParseUint(a[2], 10, 64)
				if err == nil {
					column = int(v)
				}
			}
		}
	}

	// find or create row
	ed := erow.Ed()
	erows := ed.FindERowers(filename)
	if len(erows) == 0 {
		// new row
		col, nextRow := ed.GoodColumnRowPlace()

		// for directories, place under the calling row
		if fi.IsDir() {
			col = erow.Row().Col
			nextRow = erow.Row().NextRow()
		}

		erow2 := ed.NewERowerBeforeRow(filename, col, nextRow)
		err := erow2.LoadContentClear()
		if err != nil {
			ed.Error(err)
		}
		erows = []cmdutil.ERower{erow2}
	}

	if line >= 1 {
		cmdutil.GotoLineColumnInTextArea(erows[0].Row(), line, column)
	} else {
		for _, e := range erows {
			e.Flash()
		}
	}

	return true
}
