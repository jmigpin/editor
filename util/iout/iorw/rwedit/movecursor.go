package rwedit

import (
	"errors"
	"image"
	"io"

	"github.com/jmigpin/editor/v2/util/iout/iorw"
	"github.com/jmigpin/editor/v2/util/uiutil/event"
)

func MoveCursorToPoint(ctx *Ctx, p image.Point, sel bool) {
	i := ctx.Fns.GetIndex(p)
	ctx.C.UpdateSelection(sel, i)
	// set primary copy
	if b, ok := ctx.Selection(); ok {
		ctx.Fns.SetClipboardData(event.CIPrimary, string(b))
	}
}

//----------

func MoveCursorLeft(ctx *Ctx, sel bool) error {
	ci := ctx.C.Index()
	_, size, err := iorw.ReadLastRuneAt(ctx.RW, ci)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			return err
		}
		return nil
	}
	ctx.C.UpdateSelection(sel, ci-size)
	return nil
}

func MoveCursorRight(ctx *Ctx, sel bool) error {
	ci := ctx.C.Index()
	_, size, err := iorw.ReadRuneAt(ctx.RW, ci)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			return err
		}
		return nil
	}
	ctx.C.UpdateSelection(sel, ci+size)
	return nil
}

//----------

func MoveCursorUp(ctx *Ctx, sel bool) {
	p := ctx.Fns.GetPoint(ctx.C.Index())
	p.Y -= ctx.Fns.LineHeight() - 1
	i := ctx.Fns.GetIndex(p)
	ctx.C.UpdateSelection(sel, i)
}

func MoveCursorDown(ctx *Ctx, sel bool) {
	p := ctx.Fns.GetPoint(ctx.C.Index())
	p.Y += ctx.Fns.LineHeight() + 1
	i := ctx.Fns.GetIndex(p)
	ctx.C.UpdateSelection(sel, i)
}

//----------

func MoveCursorJumpLeft(ctx *Ctx, sel bool) error {
	i, err := jumpLeftIndex(ctx)
	if err != nil {
		return err
	}
	ctx.C.UpdateSelection(sel, i)
	return nil
}
func MoveCursorJumpRight(ctx *Ctx, sel bool) error {
	i, err := jumpRightIndex(ctx)
	if err != nil {
		return err
	}
	ctx.C.UpdateSelection(sel, i)
	return nil
}

//----------

func jumpLeftIndex(ctx *Ctx) (int, error) {
	rd := ctx.LocalReader(ctx.C.Index())
	i, size, err := iorw.RuneLastIndexFn(rd, ctx.C.Index(), true, edgeOfNextWordOrNewline())
	if err != nil && !errors.Is(err, io.EOF) {
		return 0, err
	}
	return i + size, nil
}

func jumpRightIndex(ctx *Ctx) (int, error) {
	rd := ctx.LocalReader(ctx.C.Index())
	i, _, err := iorw.RuneIndexFn(rd, ctx.C.Index(), true, edgeOfNextWordOrNewline())
	if err != nil && !errors.Is(err, io.EOF) {
		return 0, err
	}
	return i, nil
}

//----------

func edgeOfNextWordOrNewline() func(rune) bool {
	first := true
	var inWord bool
	return func(ru rune) bool {
		w := iorw.IsWordRune(ru)
		if first {
			first = false
			inWord = w
		} else {
			if !inWord {
				inWord = w
				if ru == '\n' {
					return true
				}
			} else {
				if !w {
					return true
				}
			}
		}
		return false
	}
}
