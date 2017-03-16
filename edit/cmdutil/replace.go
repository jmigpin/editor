package cmdutil

import (
	"fmt"

	"github.com/jmigpin/editor/edit/toolbardata"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/ui/tautil"
)

func Replace(ed Editorer, row *ui.Row, part *toolbardata.Part) {
	a := part.Args[1:]
	if len(a) != 2 {
		ed.Error(fmt.Errorf("replace: expecting 2 arguments"))
		return
	}
	old, new := a[0].Trim(), a[1].Trim()
	tautil.Replace(row.TextArea, old, new)
}
