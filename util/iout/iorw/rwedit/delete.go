package rwedit

import "github.com/jmigpin/editor/util/iout/iorw"

func Delete(ctx *Ctx) error {
	a, b, ok := ctx.C.SelectionIndexes()
	if ok {
		ctx.C.SetSelectionOff()
	} else {
		a = ctx.C.Index()
		_, size, err := iorw.ReadRuneAt(ctx.RW, a)
		if err != nil {
			return err
		}
		b = a + size
	}
	if err := ctx.RW.OverwriteAt(a, b-a, nil); err != nil {
		return err
	}
	ctx.C.SetIndex(a)
	return nil
}
