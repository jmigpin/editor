package cmdutil

import (
	"fmt"

	"github.com/jmigpin/editor/core/toolbardata"
	"github.com/jmigpin/editor/ui/tautil"
)

func Replace(erow ERower, part *toolbardata.Part) {
	a := part.Args[1:]
	if len(a) != 2 {
		err := fmt.Errorf("replace: expecting 2 arguments")
		erow.Ed().Error(err)
		return
	}
	old, new := a[0].Str, a[1].Str
	tautil.Replace(erow.Row().TextArea, old, new)
}
