package rwedit

import (
	"bytes"

	"github.com/jmigpin/editor/util/iout/iorw"
)

func TabRight(ctx *Ctx) error {
	if !ctx.C.HaveSelection() {
		return InsertString(ctx, "\t")
	}

	a, b, newline, err := ctx.CursorSelectionLinesIndexes()
	if err != nil {
		return err
	}

	// insert at lines start
	for i := a; i < b; {
		if err := ctx.RW.Overwrite(i, 0, []byte{'\t'}); err != nil {
			return err
		}
		b += 1 // size of \t

		rd := ctx.LocalReader(i)
		u, _, err := iorw.LineEndIndex(rd, i)
		if err != nil {
			return err
		}
		i = u
	}

	// cursor index without the newline
	if newline {
		b--
	}

	ctx.C.SetSelection(a, b)
	return nil
}

func TabLeft(ctx *Ctx) error {
	a, b, newline, err := ctx.CursorSelectionLinesIndexes()
	if err != nil {
		return err
	}

	// remove from lines start
	altered := false
	for i := a; i < b; {
		s, err := ctx.RW.ReadNAtCopy(i, 1)
		if err != nil {
			return err
		}
		if bytes.ContainsAny(s, "\t ") {
			altered = true
			if err := ctx.RW.Overwrite(i, 1, nil); err != nil {
				return err
			}
			b -= 1 // 1 is length of '\t' or ' '
		}

		rd := ctx.LocalReader(i)
		u, _, err := iorw.LineEndIndex(rd, i)
		if err != nil {
			return err
		}
		i = u
	}

	// skip making the selection
	if !altered {
		return nil
	}

	// cursor index without the newline
	if newline {
		b--
	}

	ctx.C.SetSelection(a, b)
	return nil
}
