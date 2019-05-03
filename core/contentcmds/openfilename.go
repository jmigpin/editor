package contentcmds

import (
	"fmt"
	"strings"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/core/parseutil"
	"github.com/jmigpin/editor/util/iout/iorw"
)

// Detects compilers output file format <string(:int)?(:int)?>, and goes to line/column.
func OpenFilename(erow *core.ERow, index int) (bool, error) {
	ta := erow.Row.TextArea
	var rd iorw.Reader
	considerMiddle := false
	if ta.TextCursor.SelectionOn() {
		// consider only the selection
		a, b := ta.TextCursor.SelectionIndexes()
		rd = iorw.NewLimitedReader(ta.TextCursor.RW(), a, b, 0)
	} else {
		considerMiddle = true
		// limit reading
		rw := ta.TextCursor.RW()
		rd = iorw.NewLimitedReader(rw, index, index, 1000)
	}

	res, err := parseutil.ParseResource(rd, index)
	if err != nil {
		return false, err
	}

	filePos := parseutil.NewFilePosFromResource(res)

	// consider middle path (index position) if line/col are not present
	if considerMiddle && filePos.Line == 0 && filePos.Column == 0 {
		k := index - res.ExpandedMin
		if k <= 0 {
			// don't consider middle for these cases
			// k<0: index was before filename (fil^e:///a/b/c.txt)
			// k=0: index at filename start (empty string) (file://^/a/b/c.txt)
		} else {
			// index was beyond filename (/a/b/c.txt:1^:2)
			if k > len(filePos.Filename) {
				k = len(filePos.Filename)
			}
			// cut filename
			i := strings.Index(filePos.Filename[k:], string(res.PathSep))
			if i >= 0 {
				filePos.Filename = filePos.Filename[:k+i]
			}
		}
	}

	// detected it's a filename, return true from here

	// remove escapes
	filePos.Filename = parseutil.RemoveFilenameEscapes(filePos.Filename, res.Escape, res.PathSep)

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
