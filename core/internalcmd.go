package core

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/jmigpin/editor/core/toolbarparser"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

//----------

type InternalCmd struct {
	Name      string
	Fn        InternalCmdFn
	NeedsERow bool
	Detach    bool // run outside UI goroutine (care must be taken)
}

type InternalCmdFn func(args *InternalCmdArgs) error

type InternalCmdArgs struct {
	Ctx  context.Context
	Ed   *Editor
	ERow *ERow // could be nil
	Part *toolbarparser.Part
}

//----------

// cmds added via init() from "internalcmds" pkg
var InternalCmds = internalCmds{}

type internalCmds map[string]*InternalCmd

func (ic *internalCmds) Set(cmd *InternalCmd) {
	(*ic)[cmd.Name] = cmd
}

//----------

func InternalCmdFromRootTb(ed *Editor, tb *ui.Toolbar) {
	tbdata := toolbarparser.Parse(tb.Str())
	part, ok := tbdata.PartAtIndex(int(tb.CursorIndex()))
	if !ok {
		ed.Errorf("missing part at index")
		return
	}
	if len(part.Args) == 0 {
		ed.Errorf("part at index has no args")
		return
	}

	internalCmd(ed, part, nil)
}

//----------

func InternalCmdFromRowTb(erow *ERow) {
	part, ok := erow.TbData.PartAtIndex(int(erow.Row.Toolbar.CursorIndex()))
	if !ok {
		erow.Ed.Errorf("missing part at index")
		return
	}
	if len(part.Args) == 0 {
		erow.Ed.Errorf("part at index has no args")
		return
	}

	// first part cmd
	if part == erow.TbData.Parts[0] {
		if !internalCmdFromRowTbFirstPart(erow, part) {
			erow.Ed.Errorf("no cmd was run")
		}
		return
	}

	internalCmd(erow.Ed, part, erow)
}

func internalCmdFromRowTbFirstPart(erow *ERow, part *toolbarparser.Part) bool {
	a0 := part.Args[0]
	ci := erow.Row.Toolbar.CursorIndex()

	// cursor index beyond arg0
	if ci > a0.End() {
		return false
	}

	// get path up to cursor index
	a0ci := ci - a0.Pos()
	filename := a0.String()
	i := strings.Index(filename[a0ci:], string(filepath.Separator))
	if i >= 0 {
		filename = filename[:a0ci+i]
	}

	// decode filename
	filename = erow.Ed.HomeVars.Decode(filename)

	// create new row
	info := erow.Ed.ReadERowInfo(filename)
	erow2, err := NewLoadedERow(info, erow.Row.PosBelow())
	if err != nil {
		erow.Ed.Error(err)
		return true
	}

	erow2.Flash()

	// set same offset if not dir
	if erow2.Info.IsFileButNotDir() {
		ta := erow.Row.TextArea
		ta2 := erow2.Row.TextArea
		ta2.SetCursorIndex(ta.CursorIndex())
		ta2.SetRuneOffset(ta.RuneOffset())
	}

	return true
}

//----------

// erow can be nil (ex: a root toolbar cmd)
func internalCmd(ed *Editor, part *toolbarparser.Part, erow *ERow) {
	arg0 := part.Args[0].UnquotedString()
	noERowErr := func() {
		ed.Errorf("%s: no active row", arg0)
	}

	// util functions

	currentERow := func() *ERow {
		if erow != nil {
			return erow
		}
		e, ok := ed.ActiveERow()
		if ok {
			return e
		}
		return nil
	}
	run := func(detach bool, node widget.Node, fn func()) {
		if detach {
			ed.RunAsyncBusyCursor(node, func(done func()) {
				defer done()
				fn()
			})
		} else {
			fn()
		}
	}

	curERow := currentERow() // possibly != erow, could be nil

	// internal cmds
	cmd, ok := InternalCmds[arg0]
	if ok {
		ctx := context.Background() // TODO: editor ctx
		args := &InternalCmdArgs{ctx, ed, curERow, part}
		if cmd.NeedsERow && args.ERow == nil {
			noERowErr()
			return
		}

		// feedback node
		node := widget.Node(ed.UI.Root)
		if erow != nil && args.ERow == erow {
			node = erow.Row
		}

		run(cmd.Detach, node, func() {
			if cmd.NeedsERow {
				ctx, cancel := args.ERow.newInternalCmdCtx()
				defer cancel()
				args.Ctx = ctx
			}
			if err := cmd.Fn(args); err != nil {
				ed.Errorf("%v: %v", arg0, err)
			}
		})
		return
	}

	// have a plugin handle the cmd
	handled := ed.Plugins.RunToolbarCmd(curERow, part)
	if handled {
		return
	}

	// run external cmd (needs erow)
	if curERow == nil {
		noERowErr()
		return
	}
	run(false, curERow.Row, func() {
		ExternalCmd(curERow, part)
	})
}
