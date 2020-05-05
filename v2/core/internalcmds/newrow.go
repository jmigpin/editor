package internalcmds

import (
	"os"
	"path"

	"github.com/jmigpin/editor/v2/core"
)

func NewRow(args *core.InternalCmdArgs) error {
	ed := args.Ed

	p, err := os.Getwd()
	if err != nil {
		return err
	}

	rowPos := ed.GoodRowPos()

	aerow, ok := ed.ActiveERow()
	if ok {
		// stick with directory if exists, otherwise get base dir
		p2 := aerow.Info.Name()
		if aerow.Info.IsDir() {
			p = p2
		} else {
			p = path.Dir(p2)
		}

		// position after active row
		rowPos = aerow.Row.PosBelow()
	}

	info := ed.ReadERowInfo(p)

	erow, err := core.NewLoadedERow(info, rowPos)
	if err != nil {
		return err
	}
	erow.Flash()

	return nil
}
