package rwedit

func RemoveLines(ctx *Ctx) error {
	a, b, _, err := ctx.CursorSelectionLinesIndexes()
	if err != nil {
		return err
	}
	if err := ctx.RW.OverwriteAt(a, b-a, nil); err != nil {
		return err
	}
	ctx.C.SetSelectionOff()
	ctx.C.SetIndex(a)
	return nil
}
