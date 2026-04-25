package rwedit

import "github.com/jmigpin/editor/util/iout/iorw"

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

func BackspaceWord(ctx *Ctx) error {
	a, b, ok := ctx.C.SelectionIndexes()
	if ok {
		ctx.C.SetSelectionOff()
	} else {
		var err error
		b = ctx.C.Index()
		a, err = jumpLeftIndex(ctx)
		if err != nil {
			return err
		}
	}
	if err := ctx.RW.OverwriteAt(a, b-a, nil); err != nil {
		return err
	}
	ctx.C.SetIndex(a)
	return nil
}
