package rwedit

import (
	"errors"
	"image"
	"io"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/uiutil/event"
)

func MoveCursorToPoint(ctx *Ctx, p image.Point, sel bool) {
	i := ctx.Fns.GetIndex(p)
	ctx.C.SetSelectionUpdate(sel, i)
	// set primary copy
	if b, ok := ctx.Selection(); ok {
		ctx.Fns.SetClipboardData(event.CIPrimary, string(b))
	}
}

//----------

func MoveCursorLeft(ctx *Ctx, sel bool) error {
	ci := ctx.C.Index()
	_, size, err := ctx.RW.ReadLastRuneAt(ci)
	if err != nil {
		return err
	}
	ctx.C.SetSelectionUpdate(sel, ci-size)
	return nil
}

func MoveCursorRight(ctx *Ctx, sel bool) error {
	ci := ctx.C.Index()
	_, size, err := ctx.RW.ReadRuneAt(ci)
	if err != nil {
		return err
	}
	ctx.C.SetSelectionUpdate(sel, ci+size)
	return nil
}

//----------

func MoveCursorUp(ctx *Ctx, sel bool) {
	p := ctx.Fns.GetPoint(ctx.C.Index())
	p.Y -= ctx.Fns.LineHeight() - 1
	i := ctx.Fns.GetIndex(p)
	ctx.C.SetSelectionUpdate(sel, i)
}

func MoveCursorDown(ctx *Ctx, sel bool) {
	p := ctx.Fns.GetPoint(ctx.C.Index())
	p.Y += ctx.Fns.LineHeight() + 1
	i := ctx.Fns.GetIndex(p)
	ctx.C.SetSelectionUpdate(sel, i)
}

//----------

func MoveCursorJumpLeft(ctx *Ctx, sel bool) error {
	i, err := jumpLeftIndex(ctx)
	if err != nil {
		return err
	}
	ctx.C.SetSelectionUpdate(sel, i)
	return nil
}
func MoveCursorJumpRight(ctx *Ctx, sel bool) error {
	i, err := jumpRightIndex(ctx)
	if err != nil {
		return err
	}
	ctx.C.SetSelectionUpdate(sel, i)
	return nil
}

//----------

func jumpLeftIndex(ctx *Ctx) (int, error) {
	rd := ctx.LocalReader(ctx.C.Index())
	i, size, err := iorw.LastIndexFunc(rd, ctx.C.Index(), true, edgeOfNextWordOrNewline())
	if err != nil && !errors.Is(err, io.EOF) {
		return 0, err
	}
	return i + size, nil
}

func jumpRightIndex(ctx *Ctx) (int, error) {
	rd := ctx.LocalReader(ctx.C.Index())
	i, _, err := iorw.IndexFunc(rd, ctx.C.Index(), true, edgeOfNextWordOrNewline())
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
