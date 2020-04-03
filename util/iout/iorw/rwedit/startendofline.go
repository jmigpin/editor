package rwedit

import (
	"unicode"

	"github.com/jmigpin/editor/util/iout/iorw"
)

func StartOfLine(ctx *Ctx, sel bool) error {
	ci := ctx.C.Index()

	rd := ctx.LocalReader(ci)
	i, err := iorw.LineStartIndex(rd, ci)
	if err != nil {
		return err
	}

	// stop at first non blank rune from the left
	n := ci - i
	for j := 0; j < n; j++ {
		ru, _, err := iorw.ReadRuneAt(ctx.RW, i+j)
		if err != nil {
			return err
		}
		if !unicode.IsSpace(ru) {
			i += j
			break
		}
	}

	ctx.C.UpdateSelection(sel, i)
	return nil
}

func EndOfLine(ctx *Ctx, sel bool) error {
	rd := ctx.LocalReader(ctx.C.Index())
	le, newline, err := iorw.LineEndIndex(rd, ctx.C.Index())
	if err != nil {
		return err
	}
	if newline {
		le--
	}
	ctx.C.UpdateSelection(sel, le)
	return nil
}
