package rwedit

import (
	"bytes"
	"errors"
	"io"
	"unicode"

	"github.com/jmigpin/editor/util/iout/iorw"
)

func Comment(ctx *Ctx) error {
	cstrb := []byte(ctx.Fns.LineCommentStr())
	if len(cstrb) == 0 {
		return nil
	}

	a, b, newline, err := ctx.CursorSelectionLinesIndexes()
	if err != nil {
		return err
	}

	isSpaceExceptNewline := func(ru rune) bool {
		return unicode.IsSpace(ru) && ru != '\n'
	}

	// find smallest comment insertion index
	ii := 1000
	for i := a; i < b; {
		// find insertion index
		rd := ctx.LocalReader(i)
		j, _, err := iorw.IndexFunc(rd, i, false, isSpaceExceptNewline)
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}

		rd2 := ctx.LocalReader(j)
		u, _, err := iorw.LineEndIndex(rd2, j)
		if err != nil {
			return err
		}

		// ignore empty lines (j==u all spaces) and keep best
		if j != u && j-i < ii {
			ii = j - i
		}

		i = u
	}

	// insert comment
	lines := 0
	for i := a; i < b; {
		rd2 := ctx.LocalReader(i)
		u, _, err := iorw.LineEndIndex(rd2, i)
		if err != nil {
			return err
		}

		// ignore empty lines
		s, err := ctx.RW.ReadNAtCopy(i, u-i)
		if err != nil {
			return err
		}
		empty := len(bytes.TrimSpace(s)) == 0

		if !empty {
			lines++
			if err := ctx.RW.Overwrite(i+ii, 0, cstrb); err != nil {
				return err
			}
			b += len(cstrb)
			u += len(cstrb)
		}

		i = u
	}

	if lines == 0 {
		// do nothing
	} else if lines == 1 {
		// move cursor to the right due to inserted runes
		ctx.C.SetSelectionOff()
		ci := ctx.C.Index()
		if ci-a >= ii {
			ctx.C.SetIndex(ci + len(cstrb))
		}
	} else {
		// cursor index without the newline
		if newline {
			b--
		}

		ctx.C.SetSelection(a, b)
	}

	return nil
}

func Uncomment(ctx *Ctx) error {
	cstrb := []byte(ctx.Fns.LineCommentStr())
	if len(cstrb) == 0 {
		return nil
	}

	a, b, newline, err := ctx.CursorSelectionLinesIndexes()
	if err != nil {
		return err
	}

	// remove comments
	lines := 0
	ci := ctx.C.Index()
	for i := a; i < b; {
		// first non space rune (possible multiline jump)
		rd := ctx.LocalReader(i)
		j, _, err := iorw.IndexFunc(rd, i, false, unicode.IsSpace)
		if err != nil {
			break
		}
		i = j

		// remove comment runes
		if iorw.HasPrefix(ctx.RW, i, cstrb) {
			lines++
			if err := ctx.RW.Overwrite(i, len(cstrb), nil); err != nil {
				return err
			}
			b -= len(cstrb)
			if i < ci {
				// ci in between the comment string (comment len >=2)
				if i+len(cstrb) > ci {
					ci -= (i + len(cstrb)) - ci
				} else {
					ci -= len(cstrb)
				}
			}
		}

		// go to end of line
		rd2 := ctx.LocalReader(i)
		u, _, err := iorw.LineEndIndex(rd2, i)
		if err != nil {
			return err
		}
		i = u
	}

	if lines == 0 {
		// do nothing
	} else if lines == 1 {
		// move cursor to the left due to deleted runes
		ctx.C.SetSelectionOff()
		ctx.C.SetIndex(ci)
	} else {
		// cursor index without the newline
		if newline {
			b--
		}

		ctx.C.SetSelection(a, b)
	}

	return nil
}
