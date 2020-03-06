package rwedit

import (
	"errors"
	"io"
	"unicode"

	"github.com/jmigpin/editor/util/iout/iorw"
)

func AutoIndent(ctx *Ctx) error {
	ci := ctx.C.Index()

	rd1 := iorw.NewLimitedReader(ctx.RW, ci-2000, ci)
	i, err := iorw.LineStartIndex(rd1, ci)
	if err != nil {
		return err
	}

	rd := iorw.NewLimitedReader(ctx.RW, i, ci)
	j, _, err := iorw.IndexFunc(rd, i, false, unicode.IsSpace)
	if err != nil {
		if errors.Is(err, io.EOF) {
			j = ci // all spaces up to ci
		} else {
			return err
		}
	}

	// string to insert
	s, err := ctx.RW.ReadNAtCopy(i, j-i)
	if err != nil {
		return err
	}
	s2 := append([]byte{'\n'}, s...)

	// selection to overwrite
	n := 0
	if a, b, ok := ctx.C.SelectionIndexes(); ok {
		n = b - a
		ci = a
		ctx.C.SetSelectionOff()
	}

	if err := ctx.RW.Overwrite(ci, n, s2); err != nil {
		return err
	}
	ctx.C.SetIndex(ci + len(s2))
	return nil
}
