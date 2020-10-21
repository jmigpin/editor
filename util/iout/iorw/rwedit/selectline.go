package rwedit

import (
	"github.com/jmigpin/editor/util/uiutil/event"
)

func SelectLine(ctx *Ctx) error {
	ctx.C.SetSelectionOff()
	a, b, _, err := ctx.CursorSelectionLinesIndexes()
	if err != nil {
		return err
	}
	ctx.C.SetSelection(a, b)
	// set primary copy
	if b, ok := ctx.Selection(); ok {
		ctx.Fns.SetClipboardData(event.CIPrimary, string(b))
	}
	return nil
}
