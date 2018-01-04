package cmdutil

import (
	"fmt"
	"strings"

	"github.com/jmigpin/editor/core/toolbardata"
	"github.com/jmigpin/editor/ui/tautil"
)

func Find(erow ERower, part *toolbardata.Part) {
	a := part.Args[1:]
	if len(a) <= 0 {
		erow.Ed().Error(fmt.Errorf("find: expecting 1 argument"))
		return
	}
	var str string
	if len(a) == 1 {
		str = a[0].UnquotedStr()
	} else {
		// join args
		s := part.ToolbarData.Str[a[0].S:a[len(a)-1].E]
		str = strings.TrimSpace(s)
	}
	found := tautil.Find(erow.Row().TextArea, str)
	if !found {
		erow.Ed().Errorf("string not found: %q", str)
	}
}
