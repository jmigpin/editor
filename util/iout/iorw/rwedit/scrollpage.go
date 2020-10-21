package rwedit

func ScrollUp(ctx *Ctx, up bool) {
	ctx.Fns.ScrollUp(up)
}

func PageUp(ctx *Ctx, up bool) {
	ctx.Fns.PageUp(up)
}
