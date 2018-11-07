package core

import (
	"fmt"
	"strings"
)

//----------

type ContentCmd struct {
	Name string // just descriptive for err msgs and removing is wanted
	Fn   ContentCmdFn
}

type ContentCmdFn func(erow *ERow, index int) (handled bool, _ error)

//----------

var ContentCmds ContentCmdSlice

// internal cmds added via init() from "contentcmds" pkg
type ContentCmdSlice []*ContentCmd

func (ccs *ContentCmdSlice) Append(name string, fn ContentCmdFn) {
	cc := &ContentCmd{name, fn}
	*ccs = append(*ccs, cc)
}

func (ccs *ContentCmdSlice) Prepend(name string, fn ContentCmdFn) {
	cc := &ContentCmd{name, fn}
	*ccs = append([]*ContentCmd{cc}, *ccs...)
}

func (ccs *ContentCmdSlice) Remove(name string) (removed bool) {
	var a []*ContentCmd
	for _, cc := range *ccs {
		if cc.Name == name {
			removed = true
		} else {
			a = append(a, cc)
		}
	}
	*ccs = a
	return
}

//----------

func runContentCmds(erow *ERow, index int) {
	errs := []string{}
	for _, cc := range ContentCmds {
		handled, err := cc.Fn(erow, index)
		if handled {
			if err != nil {
				s := fmt.Sprintf("content cmd %q: %v", cc.Name, err)
				errs = append(errs, s)
			} else {
				// stop on first handled without error
				return
			}
		}
	}

	u := strings.Join(errs, "\n\t")
	if len(u) > 0 {
		u = "\n\t" + u
	}
	erow.Ed.Errorf("no content cmd run successfully%v", u)
}
