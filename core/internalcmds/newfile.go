package internalcmds

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmigpin/editor/core"
)

func NewFile(args *core.InternalCmdArgs) error {
	if len(args.Part.Args) != 2 {
		return fmt.Errorf("missing filename")
	}
	name := args.Part.Args[1].String()

	// erow always defined (row cmd)
	erow := args.ERow

	// directory
	dir := erow.Info.Name()
	if !erow.Info.IsDir() {
		dir = filepath.Dir(dir)
	}

	filename := filepath.Join(dir, name)

	_, err := os.Stat(filename)
	if !os.IsNotExist(err) {
		return fmt.Errorf("already exists: %v", filename)
	}
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	f.Close()

	info := args.Ed.ReadERowInfo(filename)

	rowPos := erow.Row.PosBelow()
	erow2, err := core.NewLoadedERow(info, rowPos)
	if err != nil {
		return err
	}
	erow2.Flash()

	return nil
}
