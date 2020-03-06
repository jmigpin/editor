package rwedit

func Backspace(ctx *Ctx) error {
	a, b, ok := ctx.C.SelectionIndexes()
	if ok {
		ctx.C.SetSelectionOff()
	} else {
		b = ctx.C.Index()
		_, size, err := ctx.RW.ReadLastRuneAt(b)
		if err != nil {
			return err
		}
		a = b - size
	}
	if err := ctx.RW.Overwrite(a, b-a, nil); err != nil {
		return err
	}
	ctx.C.SetIndex(a)
	return nil
}
