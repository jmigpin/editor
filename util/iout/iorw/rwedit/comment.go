package rwedit

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"unicode"

	"github.com/jmigpin/editor/util/iout/iorw"
)

func Comment(ctx *Ctx) error {
	sym := ctx.Fns.CommentLineSym()
	if sym == nil {
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
		j, _, err := iorw.RuneIndexFn(rd, i, false, isSpaceExceptNewline)
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
	ci := ctx.C.Index()
	for i := a; i < b; {
		// end of line
		rd2 := ctx.LocalReader(i)
		u, _, err := iorw.LineEndIndex(rd2, i)
		if err != nil {
			return err
		}

		// ignore empty lines
		s, err := ctx.RW.ReadFastAt(i, u-i)
		if err != nil {
			return err
		}
		empty := len(bytes.TrimSpace(s)) == 0

		if !empty {
			// end of line, last non space
			u2, size, err := iorw.RuneLastIndexFn(rd2, u, false, unicode.IsSpace)
			if err != nil {
				return err
			}
			u2 += size

			// helper func
			insert := func(k int, s string) error {
				if err := ctx.RW.OverwriteAt(k, 0, []byte(s)); err != nil {
					return err
				}
				b += len(s)
				u += len(s)
				u2 += len(s)
				if k <= ci {
					ci += len(s)
				}
				return nil
			}

			lines++
			switch t := sym.(type) {
			case string:
				if err := insert(i+ii, t); err != nil {
					return err
				}
			case [2]string:
				if err := insert(i+ii, t[0]); err != nil {
					return err
				}
				if err := insert(u2, t[1]); err != nil {
					return err
				}
			default:
				panic(fmt.Sprintf("unexpected type: %T", t))
			}
		}

		i = u
	}

	if lines == 0 {
		// do nothing
	} else if lines == 1 {
		// move cursor to the right due to inserted runes
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

func Uncomment(ctx *Ctx) error {
	sym := ctx.Fns.CommentLineSym()
	if sym == nil {
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
		j, _, err := iorw.RuneIndexFn(rd, i, false, unicode.IsSpace)
		if err != nil {
			break
		}
		i = j

		// end of line
		rd2 := ctx.LocalReader(i)
		u, _, err := iorw.LineEndIndex(rd2, i)
		if err != nil {
			return err
		}
		// end of line, last non space
		u2, size, err := iorw.RuneLastIndexFn(rd2, u, false, unicode.IsSpace)
		if err != nil {
			return err
		}
		u2 += size

		// helper func
		remove := func(k int, s string) error {
			if err := ctx.RW.OverwriteAt(k, len(s), nil); err != nil {
				return err
			}
			b -= len(s)
			u -= len(s)
			u2 -= len(s)
			if k < ci {
				// ci in between the comment string (comment len >=2)
				if k+len(s) >= ci {
					ci = k
				} else {
					ci -= len(s)
				}
			}
			return nil
		}

		switch t := sym.(type) {
		case string:
			if iorw.HasPrefix(ctx.RW, i, []byte(t)) {
				lines++
				if err := remove(i, t); err != nil {
					return err
				}
			}
		case [2]string:
			if iorw.HasPrefix(ctx.RW, i, []byte(t[0])) &&
				iorw.HasSuffix(ctx.RW, u2, []byte(t[1])) {
				lines++
				if err := remove(i, t[0]); err != nil {
					return err
				}
				if err := remove(u2-len(t[1]), t[1]); err != nil {
					return err
				}
			}
		default:
			panic(fmt.Sprintf("unexpected type: %T", t))
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
