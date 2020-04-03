package rwedit

func InsertString(ctx *Ctx, s string) error {
	n := 0
	ci := ctx.C.Index()
	if a, b, ok := ctx.C.SelectionIndexes(); ok {
		n = b - a
		ci = a
		ctx.C.SetSelectionOff()
	}
	if err := ctx.RW.OverwriteAt(ci, n, []byte(s)); err != nil {
		return err
	}
	ctx.C.SetIndex(ci + len(s))
	return nil
}
