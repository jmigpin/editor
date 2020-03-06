package rwedit

func Delete(ctx *Ctx) error {
	a, b, ok := ctx.C.SelectionIndexes()
	if ok {
		ctx.C.SetSelectionOff()
	} else {
		a = ctx.C.Index()
		_, size, err := ctx.RW.ReadRuneAt(a)
		if err != nil {
			return err
		}
		b = a + size
	}
	if err := ctx.RW.Overwrite(a, b-a, nil); err != nil {
		return err
	}
	ctx.C.SetIndex(a)
	return nil
}
