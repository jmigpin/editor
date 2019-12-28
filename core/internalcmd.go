package core

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/jmigpin/editor/core/toolbarparser"
	"github.com/jmigpin/editor/ui"
)

//----------

type InternalCmd struct {
	Name       string
	RootTbOnly bool
	Fn         func(args *InternalCmdArgs) error
}

type InternalCmdArgs struct {
	Ctx  context.Context
	Ed   *Editor
	ERow *ERow // could be nil if cmd is RootTbOnly
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
	part, ok := tbdata.PartAtIndex(int(tb.TextCursor.Index()))
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
	part, ok := erow.TbData.PartAtIndex(int(erow.Row.Toolbar.TextCursor.Index()))
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
	ci := erow.Row.Toolbar.TextCursor.Index()

	// cursor index beyond arg0
	if ci > a0.End {
		return false
	}

	// get path up to cursor index
	a0ci := ci - a0.Pos
	filename := a0.Str()
	i := strings.Index(filename[a0ci:], string(filepath.Separator))
	if i >= 0 {
		filename = filename[:a0ci+i]
	}

	// decode filename
	filename = erow.Ed.HomeVars.Decode(filename)

	// create new row
	info := erow.Ed.ReadERowInfo(filename)
	erow2, err := info.NewERow(erow.Row.PosBelow())
	if err != nil {
		erow.Ed.Error(err)
		return true
	}

	erow2.Flash()

	// set same offset if not dir
	if erow2.Info.IsFileButNotDir() {
		ta := erow.Row.TextArea
		ta2 := erow2.Row.TextArea
		ta2.TextCursor.SetIndex(ta.TextCursor.Index())
		ta2.SetRuneOffset(ta.RuneOffset())
	}

	return true
}

//----------

// erow can be nil (ex: a root toolbar cmd)
func internalCmd(ed *Editor, part *toolbarparser.Part, erow *ERow) {
	arg0 := part.Args[0].UnquotedStr()

	rootOnlyCmd := func(fn func()) {
		if erow != nil {
			ed.Errorf("%s:  root toolbar only command", arg0)
			return
		}
		fn()
	}

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

	rowCmdErr := func(fn func(*ERow) error) {
		e := currentERow()
		if e == nil {
			ed.Errorf("%s: no active row", arg0)
			return
		}
		if err := fn(e); err != nil {
			ed.Errorf("%v: %v", arg0, err)
		}
	}

	rowCmd := func(fn func(*ERow)) {
		rowCmdErr(func(e *ERow) error {
			fn(e)
			return nil
		})
	}

	// internal cmds
	cmd, ok := InternalCmds[arg0]
	if ok {
		ctx := context.Background() // TODO: editor ctx
		args := &InternalCmdArgs{ctx, ed, erow, part}
		if cmd.RootTbOnly {
			rootOnlyCmd(func() {
				if err := cmd.Fn(args); err != nil {
					ed.Errorf("%v: %v", arg0, err)
				}
			})
		} else {
			rowCmdErr(func(erow *ERow) error {
				args.ERow = erow
				args.Ctx = erow.ctx
				return cmd.Fn(args)
			})
		}
		return
	}

	// have a plugin handle the cmd
	e := currentERow() // could be nil
	handled := ed.Plugins.RunToolbarCmd(e, part)
	if handled {
		return
	}

	// run external cmd
	rowCmd(func(e *ERow) { ExternalCmd(e, part) })
}
