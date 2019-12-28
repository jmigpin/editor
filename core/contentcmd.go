package core

import (
	"context"
	"fmt"
	"strings"
)

//----------

type ContentCmd struct {
	Name string // for removal and error msgs
	Fn   ContentCmdFn
}

type ContentCmdFn func(ctx context.Context, erow *ERow, index int) (_ error, handled bool)

//----------

type contentCmds []*ContentCmd

func (ccs *contentCmds) Append(name string, fn ContentCmdFn) {
	cc := &ContentCmd{name, fn}
	*ccs = append(*ccs, cc)
}

func (ccs *contentCmds) Prepend(name string, fn ContentCmdFn) {
	cc := &ContentCmd{name, fn}
	*ccs = append([]*ContentCmd{cc}, *ccs...)
}

func (ccs *contentCmds) Remove(name string) (removed bool) {
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

// cmds added via init() from "contentcmds" pkg
var ContentCmds contentCmds

func runContentCmds(ctx context.Context, erow *ERow, index int) {
	errs := []string{}
	for _, cc := range ContentCmds {
		err, handled := cc.Fn(ctx, erow, index)
		if handled {
			if err != nil {
				s := fmt.Sprintf("%v: %v", cc.Name, err)
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
	erow.Ed.Errorf("no content cmd ran successfully%v", u)
}
