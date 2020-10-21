package internalcmds

import (
	"fmt"

	"github.com/jmigpin/editor/v2/core"
	"github.com/jmigpin/editor/v2/core/lsproto"
	"github.com/jmigpin/editor/v2/ui"
)

func LSProtoRename(args0 *core.InternalCmdArgs) error {
	erow := args0.ERow

	if !erow.Info.IsFileButNotDir() {
		return fmt.Errorf("not a file")
	}

	if erow.Row.HasState(ui.RowStateEdited | ui.RowStateFsDiffer) {
		return fmt.Errorf("row has edits, save first")
	}

	args := args0.Part.Args[1:]
	if len(args) < 1 {
		return fmt.Errorf("expecting at least 1 argument")
	}

	// new name argument "to"
	to := args[len(args)-1].UnquotedStr()

	// id offset to rename "from"
	ta := erow.Row.TextArea
	we, err := args0.Ed.LSProtoMan.TextDocumentRename(args0.Ctx, erow.Info.Name(), ta.RW(), ta.CursorIndex(), to)
	if err != nil {
		return err
	}

	wecs, err := lsproto.WorkspaceEditChanges(we)
	if err != nil {
		return err
	}

	// before patching, check all affected files are not edited
	for _, wec := range wecs {
		info, ok := args0.Ed.ERowInfo(wec.Filename)
		if !ok { // erow not open
			continue
		}
		if info.HasRowState(ui.RowStateEdited | ui.RowStateFsDiffer) {
			return fmt.Errorf("row has edits, save first: %v", info.Name())
		}
	}

	if err := lsproto.PatchWorkspaceEditChanges(wecs); err != nil {
		return err
	}

	// reload filenames
	args0.Ed.UI.RunOnUIGoRoutine(func() {
		for _, wec := range wecs {
			info, ok := args0.Ed.ERowInfo(wec.Filename)
			if !ok { // erow not open
				continue
			}
			if err := info.ReloadFile(); err != nil {
				args0.Ed.Error(err)
			}
		}
	})

	return nil
}
