package internalcmds

import (
	"fmt"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/core/lsproto"
	"github.com/jmigpin/editor/ui"
)

func LSProtoRename(args *core.InternalCmdArgs) error {
	erow, err := args.ERowOrErr()
	if err != nil {
		return err
	}

	if !erow.Info.IsFileButNotDir() {
		return fmt.Errorf("not a file")
	}

	if erow.Row.HasState(ui.RowStateEdited | ui.RowStateFsDiffer) {
		return fmt.Errorf("row has edits, save first")
	}

	args2 := args.Part.Args[1:]
	if len(args2) < 1 {
		return fmt.Errorf("expecting at least 1 argument")
	}

	// new name argument "to"
	to := args2[len(args2)-1].UnquotedString()

	// before patching, check all affected files are not edited
	prePatchFn := func(wecs []*lsproto.WorkspaceEditChange) error {
		for _, wec := range wecs {
			info, ok := args.Ed.ERowInfo(wec.Filename)
			if !ok { // erow not open
				continue
			}
			if info.HasRowState(ui.RowStateEdited | ui.RowStateFsDiffer) {
				return fmt.Errorf("row has edits, save first: %v", info.Name())
			}
		}
		return nil
	}

	// id offset to rename "from"
	ta := erow.Row.TextArea
	wecs, err := args.Ed.LSProtoMan.TextDocumentRenameAndPatch(args.Ctx, erow.Info.Name(), ta.RW(), ta.CursorIndex(), to, prePatchFn)
	if err != nil {
		return err
	}

	// reload filenames
	args.Ed.UI.RunOnUIGoRoutine(func() {
		for _, wec := range wecs {
			info, ok := args.Ed.ERowInfo(wec.Filename)
			if !ok { // erow not open
				continue
			}
			if err := info.ReloadFile(); err != nil {
				args.Ed.Error(err)
			}
		}
	})

	return nil
}
