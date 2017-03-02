package contentcmd

import (
	"os"
	"path"

	"github.com/jmigpin/editor/edit/cmdutil"
	"github.com/jmigpin/editor/ui"
)

func directory(ed cmdutil.Editori, row *ui.Row, cmd string) bool {
	p := cmd
	if !path.IsAbs(cmd) {
		tsd := ed.RowToolbarStringData(row)
		d, ok := tsd.FirstPartDirectory()
		if ok {
			p = path.Join(d, p)
		} else {
			f, ok := tsd.FirstPartFilename()
			if ok {
				p = path.Join(path.Dir(f), p)
			}
		}
	}
	fi, err := os.Stat(p)
	if err != nil {
		return false
	}
	if !fi.IsDir() {
		return false
	}
	col := ed.ActiveColumn()
	row, err = ed.FindRowOrCreateInColFromFilepath(p, col)
	if err == nil {
		row.Square.WarpPointer()
	}
	return true
}
