package core

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jmigpin/editor/core/toolbarparser"
	"github.com/jmigpin/editor/ui"
)

// cmds added via init() from "internalcmds" pkg
var InternalCmds = internalCmds{}

var noERowErr = fmt.Errorf("no active row")

//----------

type internalCmds map[string]*InternalCmd

func (ic *internalCmds) Set(cmd *InternalCmd) {
	(*ic)[cmd.Name] = cmd
}

//----------

type InternalCmd struct {
	Name string
	Fn   InternalCmdFn
}

type InternalCmdFn func(args *InternalCmdArgs) error

//----------

type InternalCmdArgs struct {
	Cmd     *InternalCmd
	Ctx     context.Context
	Ed      *Editor
	Part    *toolbarparser.Part
	optERow *ERow // can be nil
}

func (args *InternalCmdArgs) ERow() (*ERow, bool) {
	return args.optERow, args.optERow != nil
}
func (args *InternalCmdArgs) ERowOrErr() (*ERow, error) {
	erow, ok := args.ERow()
	if !ok {
		return nil, noERowErr
	}
	return erow, nil
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

	internalExternalCmd(ed, part, nil)
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

	internalExternalCmd(erow.Ed, part, erow)
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
func internalExternalCmd(ed *Editor, part *toolbarparser.Part, optERow *ERow) {

	// TODO: remove sync dependency or it will crash
	// TODO: also, need active row check here instead of internalcmd2
	//// run with busy cursor
	//node := ed.UI.RootNode
	//if optERow != nil {
	//	node = optERow.Row
	//}
	//ed.RunAsyncBusyCursor(node, func() {

	if err := internalCmd2(ed, part, optERow); err != nil {
		arg0 := part.Args[0].UnquotedString()
		ed.Errorf("%s: %w", arg0, err)
	}

	//})
}

func internalCmd2(ed *Editor, part *toolbarparser.Part, optERow *ERow) error {
	if optERow == nil {
		if ae, ok := ed.ActiveERow(); ok {
			optERow = ae
		}
	}

	if err, handled := internalCmd3(ed, part, optERow); handled {
		return err
	}

	if handled := ed.Plugins.RunToolbarCmd(optERow, part); handled {
		return nil
	}

	// run external cmd (needs erow)
	erow := optERow
	if erow == nil {
		return noERowErr
	}
	ExternalCmd(erow, part)
	return nil
}
func internalCmd3(ed *Editor, part *toolbarparser.Part, optERow *ERow) (error, bool) {
	arg0 := part.Args[0].UnquotedString()
	cmd, ok := InternalCmds[arg0]
	if !ok {
		return nil, false
	}
	ctx := context.Background()
	args := &InternalCmdArgs{cmd, ctx, ed, part, optERow}
	if args.optERow != nil {
		ctx2, cancel := args.optERow.newInternalCmdCtx()
		defer cancel()
		args.Ctx = ctx2
	}
	return cmd.Fn(args), true
}

//----------

// TODO
//// feedback node
//node := widget.Node(ed.UI.Root)
//if optERow != nil && args.ERow == optERow {
//	node = optERow.Row
//}
