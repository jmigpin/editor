package rwedit

func SelectAll(ctx *Ctx) error {
	ctx.C.SetSelection(ctx.RW.Min(), ctx.RW.Max())
	return nil
}
