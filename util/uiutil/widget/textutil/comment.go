package textutil

import (
	"bytes"
	"errors"
	"io"
	"unicode"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

func Comment(tex *widget.TextEditX) error {
	cstrb := []byte(tex.CommentLineSymbol())
	if len(cstrb) == 0 {
		return nil
	}

	tc := tex.TextCursor
	tc.BeginEdit()
	defer tc.EndEdit()

	a, b, newline, err := tc.LinesIndexes()
	if err != nil {
		return err
	}

	isSpaceExceptNewline := func(ru rune) bool {
		return unicode.IsSpace(ru) && ru != '\n'
	}

	// find smallest comment insertion index
	max := 1000
	ii := max
	for i := a; i < b; {
		// find insertion index
		rd := iorw.NewLimitedReader(tc.RW(), i, i+max)
		j, _, err := iorw.IndexFunc(rd, i, false, isSpaceExceptNewline)
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}

		u, _, err := tex.LineEndIndex(j)
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
		u, _, err := tex.LineEndIndex(i)
		if err != nil {
			return err
		}

		// ignore empty lines
		s, err := tc.RW().ReadNAtCopy(i, u-i)
		if err != nil {
			return err
		}
		empty := len(bytes.TrimSpace(s)) == 0

		if !empty {
			lines++
			if err := tc.RW().Overwrite(i+ii, 0, cstrb); err != nil {
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
		tc.SetSelectionOff()
		ci := tc.Index()
		if ci-a >= ii {
			tc.SetIndex(ci + len(cstrb))
		}
	} else {
		// cursor index without the newline
		if newline {
			b--
		}

		tc.SetSelection(a, b)
	}

	return nil
}

func Uncomment(tex *widget.TextEditX) error {
	cstrb := []byte(tex.CommentLineSymbol())
	if len(cstrb) == 0 {
		return nil
	}

	tc := tex.TextCursor
	tc.BeginEdit()
	defer tc.EndEdit()

	a, b, newline, err := tc.LinesIndexes()
	if err != nil {
		return err
	}

	// remove comments
	lines := 0
	ci := tc.Index()
	for i := a; i < b; {
		// first non space rune (possible multiline jump)
		j, _, err := tex.IndexFunc(i, false, unicode.IsSpace)
		if err != nil {
			break
		}
		i = j

		// remove comment runes
		if iorw.HasPrefix(tc.RW(), i, cstrb) {
			lines++
			if err := tc.RW().Overwrite(i, len(cstrb), nil); err != nil {
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
		u, _, err := tex.LineEndIndex(i)
		if err != nil {
			return err
		}
		i = u
	}

	if lines == 0 {
		// do nothing
	} else if lines == 1 {
		// move cursor to the left due to deleted runes
		tc.SetSelectionOff()
		tc.SetIndex(ci)
	} else {
		// cursor index without the newline
		if newline {
			b--
		}

		tc.SetSelection(a, b)
	}

	return nil
}
