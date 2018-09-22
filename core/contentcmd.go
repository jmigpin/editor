package core

import (
	"fmt"
	"strings"
)

/*
Content commands are registered in a 3rd package.
Ex: in $project/editor.go located at "github.com/jmigpin/editor":

	import "github.com/jmigpin/editor/core"
	import _ "github.com/jmigpin/editor/core/contentcmds"

The second line import runs the package "init()" functions that makes the registrations
*/

type ContentCmdFn func(erow *ERow, index int) (handled bool, _ error)

var contentCmds []ContentCmdFn

func RegisterContentCmd(fn ContentCmdFn) {
	contentCmds = append(contentCmds, fn)
}

func RunContentCmds(erow *ERow, index int) {
	errs := []string{}
	for i, fn := range contentCmds {
		handled, err := fn(erow, index)
		if err != nil {
			if handled {
				s := fmt.Sprintf("\tcmd %v: %v", i, err)
				errs = append(errs, s)
			}
		}
		// stop on first handled without error
		if handled && err == nil {
			return
		}
	}

	u := strings.Join(errs, "\n")
	if len(u) > 0 {
		u = "\n" + u
	}

	erow.Ed.Errorf("no content cmd run successfully%v", u)
}
