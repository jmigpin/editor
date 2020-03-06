package rwedit

func StartOfString(ctx *Ctx, sel bool) {
	ctx.C.SetSelectionUpdate(sel, 0)
}

func EndOfString(ctx *Ctx, sel bool) {
	ctx.C.SetSelectionUpdate(sel, ctx.RW.Max())
}
