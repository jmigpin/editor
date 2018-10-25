package contentcmds

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/core/parseutil"
)

// Detects compilers output file format <string(:int)?(:int)?>, and goes to line/column.
func filename(erow *core.ERow, index int) (bool, error) {
	ta := erow.Row.TextArea

	var str string
	considerMiddle := false
	if ta.TextCursor.SelectionOn() {
		s, err := ta.TextCursor.Selection()
		if err != nil {
			return false, err
		}
		str = string(s)
	} else {
		considerMiddle = true
		max := 500
		str = ta.Str()
		li := parseutil.ExpandLastIndexOfFilenameFmt(str[:index], max)
		if li < 0 {
			return false, fmt.Errorf("failed to expand filename to the left")
		}

		// adjust to the found left index
		str = str[li:]
		index -= li

		if len(str) > max {
			str = str[:max]
		}
	}

	filePos, err := parseutil.ParseFilePos(str)
	if err != nil {
		return false, err
	}

	// detected it's a filename, return true from here

	// consider middle path (index position) if line and column are not present
	if considerMiddle && filePos.Line == 0 && filePos.Column == 0 {
		// if filename detection is short, update index
		if len(filePos.Filename) < index {
			index = len(filePos.Filename)
		}

		i := strings.Index(filePos.Filename[index:], string(filepath.Separator))
		if i >= 0 {
			filePos.Filename = filePos.Filename[:index+i]
		}
	}

	// unescape filename
	filePos.Filename = parseutil.UnescapeString(filePos.Filename)

	// find full filename
	filename, fi, ok := core.FindFileInfo(filePos.Filename, erow.Info.Dir())
	if !ok {
		return true, fmt.Errorf("fileinfo not found: %v", filePos.Filename)
	}
	filePos.Filename = filename

	// place new under the calling row
	rowPos := erow.Row.PosBelow()
	// if calling erow is dir, and new is not dir, place at a good place
	if erow.Info.IsDir() && !fi.IsDir() {
		rowPos = erow.Ed.GoodRowPos()
	}

	conf := &core.OpenFileERowConfig{
		FilePos:               filePos,
		RowPos:                rowPos,
		FlashVisibleOffsets:   true,
		NewIfNotExistent:      true,
		NewIfOffsetNotVisible: true,
	}
	core.OpenFileERow(erow.Ed, conf)

	return true, nil
}
