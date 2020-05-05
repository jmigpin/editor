package rwedit

import "github.com/jmigpin/editor/v2/util/iout/iorw"

func Backspace(ctx *Ctx) error {
	a, b, ok := ctx.C.SelectionIndexes()
	if ok {
		ctx.C.SetSelectionOff()
	} else {
		b = ctx.C.Index()
		_, size, err := iorw.ReadLastRuneAt(ctx.RW, b)
		if err != nil {
			return err
		}
		a = b - size
	}
	if err := ctx.RW.OverwriteAt(a, b-a, nil); err != nil {
		return err
	}
	ctx.C.SetIndex(a)
	return nil
}
