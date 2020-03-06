package rwedit

func Undo(ctx *Ctx) error {
	return ctx.Fns.Undo()
}
func Redo(ctx *Ctx) error {
	return ctx.Fns.Redo()
}
