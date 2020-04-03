package rwedit

import "github.com/jmigpin/editor/util/iout/iorw"

func DuplicateLines(ctx *Ctx) error {
	a, b, newline, err := ctx.CursorSelectionLinesIndexes()
	if err != nil {
		return err
	}

	s0, err := ctx.RW.ReadFastAt(a, b-a)
	if err != nil {
		return err
	}
	s := iorw.MakeBytesCopy(s0)

	c := b
	if !newline {
		s = append([]byte{'\n'}, s...)
		c++
	}

	if err := ctx.RW.OverwriteAt(b, 0, s); err != nil {
		return err
	}

	// cursor index without the newline
	d := b + len(s)
	if newline && len(s) > 0 && s[len(s)-1] == '\n' {
		d--
	}

	ctx.C.SetSelection(c, d)
	return nil
}
