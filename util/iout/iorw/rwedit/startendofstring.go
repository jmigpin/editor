package rwedit

func StartOfString(ctx *Ctx, sel bool) {
	ctx.C.UpdateSelection(sel, 0)
}

func EndOfString(ctx *Ctx, sel bool) {
	ctx.C.UpdateSelection(sel, ctx.RW.Max())
}
