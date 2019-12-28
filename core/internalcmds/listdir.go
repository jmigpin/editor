package internalcmds

import (
	"fmt"

	"github.com/jmigpin/editor/core"
)

func ListDir(args0 *core.InternalCmdArgs) error {
	erow := args0.ERow
	part := args0.Part

	if !erow.Info.IsDir() {
		return fmt.Errorf("not a directory")
	}

	tree, hidden := false, false

	args := part.Args[1:]
	for _, a := range args {
		s := a.UnquotedStr()
		switch s {
		case "-sub":
			tree = true
		case "-hidden":
			hidden = true
		}
	}

	core.ListDirERow(erow, erow.Info.Name(), tree, hidden)

	return nil
}
