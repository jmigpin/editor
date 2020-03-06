package rwedit

import (
	"github.com/jmigpin/editor/util/iout/iorw"
)

func Replace(ctx *Ctx, old, new string) (bool, error) {
	if old == "" {
		return false, nil
	}

	oldb := []byte(old)
	newb := []byte(new)

	a, b, ok := ctx.C.SelectionIndexes()
	if !ok {
		a = ctx.RW.Min()
		b = ctx.RW.Max()
	}

	ci, replaced, err := replace2(ctx, oldb, newb, a, b)
	if err != nil {
		return replaced, err
	}
	ctx.C.SetIndex(ci)
	return replaced, nil
}

func replace2(ctx *Ctx, oldb, newb []byte, a, b int) (int, bool, error) {
	ci := ctx.C.Index()
	replaced := false
	for a < b {
		rd := iorw.NewLimitedReader(ctx.RW, a, b)
		i, err := iorw.Index(rd, a, oldb, false)
		if err != nil {
			return ci, replaced, err
		}
		if i < 0 {
			return ci, replaced, nil
		}
		if err := ctx.RW.Overwrite(i, len(oldb), newb); err != nil {
			return ci, replaced, err
		}
		replaced = true
		d := -len(oldb) + len(newb)
		b += d
		a = i + len(newb)

		if i < ci {
			ci += d
			if ci < i {
				ci = i
			}
		}
	}
	return ci, replaced, nil
}
