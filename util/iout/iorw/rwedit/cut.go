package rwedit

import (
	"github.com/jmigpin/editor/util/uiutil/event"
)

func Cut(ctx *Ctx) error {
	a, b, ok := ctx.C.SelectionIndexes()
	if !ok {
		return nil
	}

	s, err := ctx.RW.ReadNAtCopy(a, b-a)
	if err != nil {
		return err
	}
	ctx.Fns.SetClipboardData(event.CIClipboard, string(s))

	if err := ctx.RW.Overwrite(a, b-a, nil); err != nil {
		return err
	}
	ctx.C.SetSelectionOff()
	ctx.C.SetIndex(a)
	return nil
}
