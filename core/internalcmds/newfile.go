package internalcmds

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/core/toolbarparser"
)

func NewFile(args *core.InternalCmdArgs) error {
	name, createParents, err := newFileArgs(args.Part)
	if err != nil {
		return err
	}

	erow, err := args.ERowOrErr()
	if err != nil {
		return err
	}

	// directory
	dir := erow.Info.Name()
	if !erow.Info.IsDir() {
		dir = filepath.Dir(dir)
	}

	filename := name
	if !filepath.IsAbs(name) {
		filename = filepath.Join(dir, name)
	}

	if err := createNewFile(filename, createParents); err != nil {
		return err
	}

	info := args.Ed.ReadERowInfo(filename)

	rowPos := erow.Row.PosBelow()
	erow2, err := core.NewLoadedERow(info, rowPos)
	if err != nil {
		return err
	}
	erow2.Flash()

	return nil
}

//----------

func newFileArgs(part *toolbarparser.Part) (string, bool, error) {
	args := part.ArgsStrings()
	if len(args) == 3 && args[1] == "-p" {
		return args[2], true, nil
	}
	if len(args) == 2 && args[1] != "-p" {
		return args[1], false, nil
	}
	return "", false, fmt.Errorf("missing filename or invalid arguments")
}

func createNewFile(filename string, createParents bool) error {
	dir := filepath.Dir(filename)
	if createParents {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %q: %w", dir, err)
		}
	}
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("already exists: %v", filename)
		}
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %q: %w", dir, err)
		}
		return fmt.Errorf("create %q: %w", filename, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close %q: %w", filename, err)
	}
	return nil
}
